package main

//TODO: добавить функцию по умолчанию сохранения даты и времени исходного файла

import (
	"flag"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"

	"github.com/nfnt/resize"
)

func printHelp() {
	fmt.Println("Использование:")
	fmt.Println("  resize_image -input <input_dir> -width <width> [-r] [-quality <1-100>] [-threads <num>]")
	fmt.Println("Параметры:")
	fmt.Println("  -input   Путь к директории с изображениями (обязательный, если не указана текущая директория)")
	fmt.Println("  -width   Новая ширина изображений (обязательный)")
	fmt.Println("  -r       Перезаписать входные файлы (если указано)")
	fmt.Println("  -newdate Текущая дата файлов (если указано) (по умолчанию оставляем оригинальную дату файла)")
	fmt.Println("  -quality Уровень качества выходных изображений (по умолчанию 100)")
	fmt.Println("  -threads Количество параллельных потоков (по умолчанию 2)")
	fmt.Println("  -help    Показать это сообщение")
}

func processImage(inputPath string, newWidth uint, rewrite bool, newdate bool, quality int, wg *sync.WaitGroup, sem chan struct{}, stats *Statistics, mu *sync.Mutex) {
	defer wg.Done()
	sem <- struct{}{}        // Захватываем семафор
	defer func() { <-sem }() // Освобождаем семафор

	// Открываем исходное изображение
	inputFile, err := os.Open(inputPath)
	if err != nil {
		fmt.Println("Ошибка при открытии файла:", err)
		return
	}
	defer inputFile.Close()

	// Получаем информацию о размере входного файла
	inputFileInfo, err := inputFile.Stat()
	if err != nil {
		fmt.Println("Ошибка при получении информации о файле:", err)
		return
	}
	inputFileSize := inputFileInfo.Size()

	// Декодируем изображение
	img, err := jpeg.Decode(inputFile)
	if err != nil {
		fmt.Println("Ошибка при декодировании изображения:", err)
		return
	}

	// Изменяем размер изображения с сохранением пропорций
	resizedImg := resize.Resize(newWidth, 0, img, resize.Lanczos3)

	var outputFile *os.File
	var outputPath string

	// Если флаг перезаписи установлен, используем входной файл как выходной
	if rewrite {
		outputPath = inputPath
		outputFile = inputFile // Используем входный файл как выходный
	} else {
		outputPath = inputPath[:len(inputPath)-len(filepath.Ext(inputPath))] + "_r" + filepath.Ext(inputPath)
		outputFile, err = os.Create(outputPath)
		if err != nil {
			fmt.Println("Ошибка при создании файла:", err)
			return
		}
		defer outputFile.Close()
	}

	// Если перезаписываем, создаем новый файл
	if rewrite {
		outputFile, err = os.Create(inputPath)
		if err != nil {
			fmt.Println("Ошибка при создании файла для перезаписи:", err)
			return
		}
		defer outputFile.Close()
	}

	// Кодируем и сохраняем изображение в формате JPG с указанным качеством
	jpegOptions := jpeg.Options{Quality: quality}
	err = jpeg.Encode(outputFile, resizedImg, &jpegOptions)
	if err != nil {
		fmt.Println("Ошибка при сохранении изображения:", err)
		return
	}

	// Устанавливаем время модификации входного файла в выходном файле, по умолчанию оставляем оригинальную дату
	if !newdate {
		err = os.Chtimes(outputPath, inputFileInfo.ModTime(), inputFileInfo.ModTime())
		if err != nil {
			fmt.Println("Ошибка при установке временных меток:", err)
			return
		}
	}

	// Получаем информацию о размере выходного файла
	outputFileInfo, err := outputFile.Stat()
	if err != nil {
		fmt.Println("Ошибка при получении информации о выходном файле:", err)
		return
	}
	outputFileSize := outputFileInfo.Size()

	// Обновляем статистику
	mu.Lock()
	stats.TotalInputSize += inputFileSize
	stats.TotalOutputSize += outputFileSize
	stats.ProcessedFiles++
	mu.Unlock()

	// Выводим прогресс
	fmt.Printf("Обработано: %s, Входной размер: %d байт, Выходной размер: %d байт\n", inputPath, inputFileSize, outputFileSize)
}

type Statistics struct {
	TotalInputSize  int64
	TotalOutputSize int64
	ProcessedFiles  int
}

func main() {
	// Определяем флаги
	inputDir := flag.String("input", "", "Путь к директории с изображениями")
	newWidth := flag.Uint("width", 0, "Новая ширина изображений (обязательный)")
	rewrite := flag.Bool("r", false, "Перезаписать входные файлы")
	newdate := flag.Bool("newdate", false, "установить текущую дату файла")
	quality := flag.Int("quality", 100, "Уровень качества выходных изображений (1-100)")
	threads := flag.Int("threads", 2, "Количество параллельных потоков (по умолчанию 2)")

	// Обработка флагов
	flag.Parse()

	// Проверка на наличие флага помощи
	if *newWidth == 0 {
		printHelp()
		return
	}

	// Если inputDir пустой, используем текущую директорию
	if *inputDir == "" {
		var err error
		*inputDir, err = os.Getwd() // Получаем текущую рабочую директорию
		if err != nil {
			fmt.Println("Ошибка при получении текущей директории:", err)
			return
		}
	}

	// Создаем семафор для ограничения количества параллельных потоков
	sem := make(chan struct{}, *threads)

	var wg sync.WaitGroup
	var mu sync.Mutex
	stats := &Statistics{}

	// Получаем список всех файлов .jpg и .jpeg в указанной директории
	err := filepath.Walk(*inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Проверяем, что это файл и он находится в текущей директории
		if !info.IsDir() && (filepath.Ext(path) == ".jpg" || filepath.Ext(path) == ".jpeg") && filepath.Dir(path) == *inputDir {
			wg.Add(1)
			go processImage(path, *newWidth, *rewrite, *newdate, *quality, &wg, sem, stats, &mu)
		}
		return nil
	})

	if err != nil {
		fmt.Println("Ошибка при обходе директории:", err)
		return
	}

	// Ждем завершения всех горутин
	wg.Wait()

	// Выводим общую статистику в мегабайтах
	fmt.Printf("\nCтатистика:\n")
	fmt.Printf("Обработано файлов: %d\n", stats.ProcessedFiles)
	fmt.Printf("Общий входной размер: %.2f МБ\n", float64(stats.TotalInputSize)/1024/1024)
	fmt.Printf("Общий выходной размер: %.2f МБ\n", float64(stats.TotalOutputSize)/1024/1024)
	fmt.Printf("Общее сокращение размера: %.2f МБ, %.2f%%\n", (float64(stats.TotalInputSize)-float64(stats.TotalOutputSize))/1024/1024, float64(stats.TotalInputSize-stats.TotalOutputSize)/float64(stats.TotalInputSize)*100)

	fmt.Println("Обработка завершена.")
}

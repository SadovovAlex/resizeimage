package main

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
	fmt.Println("  resize_image -input <input_dir> -maxwidth <width> [-r] [-quality <1-100>] [-threads <num>]")
	fmt.Println("Параметры:")
	fmt.Println("  -input   Путь к директории с изображениями (обязательный, если не указана текущая директория)")
	fmt.Println("  -maxwidth   Новая ширина изображений (обязательный)")
	fmt.Println("  -r       Перезаписать входные файлы (если указано)")
	fmt.Println("  -newdate Текущая дата файлов (если указано) (по умолчанию оставляем оригинальную дату файла)")
	fmt.Println("  -quality Уровень качества выходных изображений (по умолчанию 100)")
	fmt.Println("  -threads Количество параллельных потоков (по умолчанию 2)")
	fmt.Println("  -help    Показать это сообщение")
}

func processImage(inputPath string, newWidth uint, rewrite bool, newdate bool, quality int, wg *sync.WaitGroup, sem chan struct{}, stats *Statistics, mu *sync.Mutex) {
	cCutDirName := 50
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

	// Получаем размеры изображения
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Проверяем, если ширина изображения меньше newWidth, пропускаем обработку
	if imgWidth <= int(newWidth) {
		if len(inputPath) > cCutDirName {
			fmt.Printf(
				"Processed: ...%s, width (%d*%d) < maxwidth %d,  skip\n",
				inputPath[cCutDirName:], imgWidth, imgHeight, newWidth,
			)
		} else {
			fmt.Printf(
				"Processed: %s, width (%d*%d) < maxwidth %d,  skip\n",
				inputPath, imgWidth, imgHeight, newWidth,
			)
		}
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
	if len(inputPath) > cCutDirName {
		fmt.Printf(
			"Processed: ...%s, %.2f->%.2fMb, optimize:%.2f%%, %.2fMb free\n",
			inputPath[cCutDirName:],
			float64(inputFileSize)/1024/1024,
			float64(outputFileSize)/1024/1024,
			float64(inputFileSize-outputFileSize)/float64(inputFileSize)*100,
			float64(inputFileSize-outputFileSize)/1024/1024,
		)
	} else {
		fmt.Printf(
			"Processed: %s,In/Out:%.2f/%.2fMb, optimize:%.2f%%, %.2fMb free\n",
			inputPath,
			float64(inputFileSize)/1024/1024,
			float64(outputFileSize)/1024/1024,
			float64(inputFileSize-outputFileSize)/float64(inputFileSize)*100,
			float64(inputFileSize-outputFileSize)/1024/1024,
		)
	}
	//fmt.Printf("Общее сокращение размера: %.2f МБ, %.2f%%\n", (float64(stats.TotalInputSize)-float64(stats.TotalOutputSize))/1024/1024, float64(stats.TotalInputSize-stats.TotalOutputSize)/float64(stats.TotalInputSize)*100)

}

type Statistics struct {
	TotalInputSize  int64
	TotalOutputSize int64
	ProcessedFiles  int
}

// Функция для чтения директории с использованием кодировки UTF-8
func readDirUTF8(path string) ([]os.DirEntry, error) {
	d, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	names, err := d.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	return names, nil
}

func main() {
	// Определяем флаги
	inputDir := flag.String("input", "", "Путь к директории с изображениями")
	newWidth := flag.Uint("maxwidth", 0, "Новая ширина изображений (обязательный)")
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
	fmt.Println("Чтение директории:", *inputDir)

	cleanPath, err := filepath.Abs(*inputDir)
	if err != nil {
		fmt.Println("Ошибка при получении абсолютного пути:", err)
		return
	}
	fmt.Println("Clean директории:", cleanPath)

	files, err := os.ReadDir(cleanPath)
	if err != nil {
		// Если возникла ошибка при чтении директории, попробуем использовать кодировку UTF-8
		files, err = readDirUTF8(cleanPath)
		if err != nil {
			fmt.Println("Ошибка при чтении директории:", err)
			return
		}
	}

	for _, file := range files {
		if !file.IsDir() && (filepath.Ext(file.Name()) == ".jpg" || filepath.Ext(file.Name()) == ".JPG" || filepath.Ext(file.Name()) == ".jpeg") {
			filePath := filepath.Join(cleanPath, file.Name())
			wg.Add(1)
			go processImage(filePath, *newWidth, *rewrite, *newdate, *quality, &wg, sem, stats, &mu)
		}
	}

	// Ждем завершения всех горутин
	wg.Wait()

	// Выводим общую статистику в мегабайтах
	fmt.Printf("\nStatistics:\n")
	fmt.Printf("Processed Files:%d\n", stats.ProcessedFiles)
	fmt.Printf("Total Input:%.2fMb\n", float64(stats.TotalInputSize)/1024/1024)
	fmt.Printf("Total Output:%.2fMb\n", float64(stats.TotalOutputSize)/1024/1024)
	fmt.Printf("Total Savings: %.2fMb, %.2f%% of original size\n", (float64(stats.TotalInputSize)-float64(stats.TotalOutputSize))/1024/1024, float64(stats.TotalInputSize-stats.TotalOutputSize)/float64(stats.TotalInputSize)*100)

	fmt.Println("Prcessing done.")
}

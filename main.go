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
	fmt.Println("  resize_image -input <input_dir> -width <width> [-r] [-quality <1-100>] [-threads <num>]")
	fmt.Println("Параметры:")
	fmt.Println("  -input   Путь к директории с изображениями (обязательный)")
	fmt.Println("  -width   Новая ширина изображений (обязательный)")
	fmt.Println("  -r       Перезаписать входные файлы (если указано)")
	fmt.Println("  -quality Уровень качества выходных изображений (по умолчанию 100)")
	fmt.Println("  -threads Количество параллельных потоков (по умолчанию 1)")
	fmt.Println("  -help    Показать это сообщение")
}

func processImage(inputPath string, newWidth uint, rewrite bool, quality int, wg *sync.WaitGroup, sem chan struct{}) {
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
		outputFile = inputFile // Это неправильно, нужно создать новый файл
	} else {
		outputPath = inputPath[:len(inputPath)-len(filepath.Ext(inputPath))] + "_resized" + filepath.Ext(inputPath)
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

	// Получаем информацию о размере выходного файла
	outputFileInfo, err := outputFile.Stat()
	if err != nil {
		fmt.Println("Ошибка при получении информации о выходном файле:", err)
		return
	}
	outputFileSize := outputFileInfo.Size()

	// Выводим статистику по сокращению размера файла
	fmt.Printf("Обработано: %s\n", inputPath)
	fmt.Printf("Размер входного файла: %d байт\n", inputFileSize)
	fmt.Printf("Размер выходного файла: %d байт\n", outputFileSize)
	fmt.Printf("Сокращение размера: %.2f%%\n", float64(inputFileSize-outputFileSize)/float64(inputFileSize)*100)
}

func main() {
	// Определяем флаги
	inputDir := flag.String("input", "", "Путь к директории с изображениями")
	newWidth := flag.Uint("width", 0, "Новая ширина изображений")
	rewrite := flag.Bool("r", false, "Перезаписать входные файлы")
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

	// Получаем список всех файлов .jpg и .jpeg в указанной директории
	err := filepath.Walk(*inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".jpg" || filepath.Ext(path) == ".jpeg") {
			wg.Add(1)
			go processImage(path, *newWidth, *rewrite, *quality, &wg, sem)
		}
		return nil
	})

	if err != nil {
		fmt.Println("Ошибка при обходе директории:", err)
		return
	}

	// Ждем завершения всех горутин
	wg.Wait()
	fmt.Println("Обработка завершена.")
}

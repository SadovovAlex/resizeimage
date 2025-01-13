package main

import (
	"flag"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/k0kubun/go-ansi"
	"github.com/nfnt/resize"
	"github.com/schollz/progressbar/v3"
)

func printHelp() {
	fmt.Println("Использование:")
	fmt.Println("  resize_image -input <input_dir> -maxwidth <width> [-rw -R] [-quality <1-100>] [-threads <num>]")
	fmt.Println("Параметры:")
	fmt.Println("  -input   Путь к директории с изображениями (обязательный, если не указана текущая директория)")
	fmt.Println("  -R   Искать в поддиректоиях")
	fmt.Println("  -maxwidth   Новая ширина изображений (обязательный)")
	fmt.Println("  -rw       Перезаписать входные файлы (если указано)")
	fmt.Println("  -newdate Текущая дата файлов (если указано) (по умолчанию оставляем оригинальную дату файла)")
	fmt.Println("  -quality Уровень качества выходных изображений (по умолчанию 100)")
	fmt.Println("  -threads Количество параллельных потоков (по умолчанию 2)")
	fmt.Println("  -help    Показать это сообщение")
	fmt.Println("  Пример: Сжать все файлы в директории с:\foto[-input], переписать оригиналы(заменить)[-rw], во всех поддиректориях[-R], размером более 2048[-maxwidth], качество 80[-quality 80], в 10 потоков[-threads 10]")
	fmt.Println("  resize_image -input C:\foto -maxwidth 2048 -rw -R -quality 80 -threads 10")
}

func processImage(inputPath string, newWidth uint, rewrite bool, newdate bool, quality int, wg *sync.WaitGroup, sem chan struct{}, stats *Statistics, mu *sync.Mutex, goroutineID int, bar *progressbar.ProgressBar) {
	//cCutDirName := 50
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

	// Обновляем прогресс-бар
	//bar.Describe(inputFile.Name())
	bar.Add(1)

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
		//fmt.Printf("[%d]Ошибка при декодировании изображения: %v\n", goroutineID, err)

		return
	}

	// Получаем размеры изображения
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	//imgHeight := bounds.Dy()

	// Проверяем, если ширина изображения меньше newWidth, пропускаем обработку
	if imgWidth <= int(newWidth) {
		/*if len(inputPath) > cCutDirName {
			fmt.Printf(
				"Skip: ...%s, width (%d*%d) <= maxwidth %d\n",
				inputPath[len(inputPath)-cCutDirName:], imgWidth, imgHeight, newWidth,
			)
		} else {
			fmt.Printf(
				"Skip: %s, width (%d*%d) <= maxwidth %d\n",
				inputPath, imgWidth, imgHeight, newWidth,
			)
		}
		*/
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
	/*
		if len(inputPath) > cCutDirName {
			fmt.Printf(
				"Processed: ...%s, %.2f->%.2fMb, optimize:%.2f%%, %.2fMb free\n",
				inputPath[len(inputPath)-cCutDirName:],
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

	*/

}

type Statistics struct {
	TotalInputSize  int64
	TotalOutputSize int64
	ProcessedFiles  int
}

func main() {
	// Определяем флаги
	inputDir := flag.String("input", "", "Путь к директории с изображениями")
	recursion := flag.Bool("R", false, "Искать в поддиректориях")
	newWidth := flag.Uint("maxwidth", 0, "Новая ширина изображений (обязательный)")
	rewrite := flag.Bool("rw", false, "Перезаписать входные файлы")
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

	// Поиск в текущей директории
	var count int
	var totalSize int64
	var jpgFiles []string
	fmt.Println("Find files:", *inputDir)
	if *recursion {
		count, totalSize, jpgFiles = findJpgFiles(*inputDir, true)
	} else {
		count, totalSize, jpgFiles = findJpgFiles(*inputDir, false)
	}
	fmt.Printf("Найдено %d файлов JPG, размер: %.2fМб\n", count, float64(totalSize)/1024/1024)

	// Создаем прогресс-бар
	//bar := progressbar.Default(int64(len(jpgFiles)), "Processing files" )
	bar := progressbar.NewOptions(int(len(jpgFiles)),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetMaxDetailRow(3),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        " ",
			AltSaucerHead: "[yellow]<[reset]",
			SaucerHead:    "[yellow]-[reset]",
			SaucerPadding: "[white]•",
			BarStart:      "[blue]|[reset]",
			BarEnd:        "[blue]|[reset]",
		}),
	)

	for i, filePath := range jpgFiles {
		//fmt.Println(filePath)
		wg.Add(1)
		go processImage(filePath, *newWidth, *rewrite, *newdate, *quality, &wg, sem, stats, &mu, i, bar)
	}

	/*
		for _, file := range files {
			if !file.IsDir() && (filepath.Ext(file.Name()) == ".jpg" || filepath.Ext(file.Name()) == ".JPG" || filepath.Ext(file.Name()) == ".jpeg") {
				filePath := filepath.Join(cleanPath, file.Name())
				wg.Add(1)
				go processImage(filePath, *newWidth, *rewrite, *newdate, *quality, &wg, sem, stats, &mu)
			}
		}
	*/

	// Ждем завершения всех горутин
	wg.Wait()

	// Выводим общую статистику в мегабайтах
	fmt.Println("\n╔═════════════════════════════════════════════════════╗")
	fmt.Printf("║ Processed Files: %d\n", stats.ProcessedFiles)
	fmt.Printf("║ Total Input: %.2fMb Total Output: %.2fMb\n", float64(stats.TotalInputSize)/1024/1024, float64(stats.TotalOutputSize)/1024/1024)
	fmt.Printf("║ Total Space Save: %.2fMb, %.2f%% of original size\n",
		(float64(stats.TotalInputSize)-float64(stats.TotalOutputSize))/1024/1024,
		float64(stats.TotalInputSize-stats.TotalOutputSize)/float64(stats.TotalInputSize)*100)
	fmt.Println("╚═════════════════════════════════════════════════════╝")
	fmt.Println("Processing done.")
}
func findJpgFiles(dir string, recursive bool) (int, int64, []string) {
	var jpgFiles []string
	var totalSize int64 = 0
	var count int = 0

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Printf("Ошибка при чтении директории %s: %v", dir, err)
		return count, totalSize, jpgFiles
	}

	for _, file := range files {
		if file.IsDir() && recursive {
			// Рекурсивно вызываем функцию для поддиректории
			subCount, subSize, subFiles := findJpgFiles(filepath.Join(dir, file.Name()), recursive)
			count += subCount
			totalSize += subSize
			jpgFiles = append(jpgFiles, subFiles...)
		} else if !file.IsDir() && strings.HasSuffix(file.Name(), ".jpg") {
			filePath := filepath.Join(dir, file.Name())
			if runtime.GOOS == "windows" {
				// Преобразуем путь к файлу в кодировку Windows
				filePath = convertToWindowsPath(filePath)
			}
			jpgFiles = append(jpgFiles, filePath)
			totalSize += file.Size()
			count++
		}
	}

	return count, totalSize, jpgFiles
}

func convertToWindowsPath(path string) string {
	// Преобразуем путь к файлу в кодировку Windows
	return string([]byte(path))
}

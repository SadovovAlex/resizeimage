package main

import (
	"flag"
	"fmt"
	"image/jpeg"
	"os"

	"github.com/nfnt/resize"
)

func printHelp() {
	fmt.Println("Использование:")
	fmt.Println("  resize_image -input <input.jpg> -output <output.jpg> -width <width> [-r] [-quality <1-100>]")
	fmt.Println("Параметры:")
	fmt.Println("  -input   Путь к входному изображению (обязательный)")
	fmt.Println("  -output  Путь к выходному изображению (обязательный, если не используется -r)")
	fmt.Println("  -width   Новая ширина изображения (обязательный)")
	fmt.Println("  -r       Перезаписать входной файл (если указан)")
	fmt.Println("  -quality Уровень качества выходного изображения (по умолчанию 100)")
	fmt.Println("  -help    Показать это сообщение")
}

func main() {
	// Определяем флаги
	inputPath := flag.String("input", "", "Путь к входному изображению")
	outputPath := flag.String("output", "", "Путь к выходному изображению")
	newWidth := flag.Uint("width", 0, "Новая ширина изображения")
	rewrite := flag.Bool("r", false, "Перезаписать входной файл")
	quality := flag.Int("quality", 100, "Уровень качества выходного изображения (1-100)")

	// Обработка флагов
	flag.Parse()

	// Проверка на наличие флага помощи
	if *inputPath == "" || *newWidth == 0 {
		printHelp()
		return
	}

	// Открываем исходное изображение
	inputFile, err := os.Open(*inputPath)
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
	resizedImg := resize.Resize(*newWidth, 0, img, resize.Lanczos3)

	var outputFile *os.File

	// Если флаг перезаписи установлен, создаем выходной файл с тем же именем
	if *rewrite {
		outputFile, err = os.Create(*inputPath)
		if err != nil {
			fmt.Println("Ошибка при создании файла для перезаписи:", err)
			return
		}
		defer outputFile.Close()
	} else {
		// Создаем выходной файл
		outputFile, err = os.Create(*outputPath)
		if err != nil {
			fmt.Println("Ошибка при создании файла:", err)
			return
		}
		defer outputFile.Close()
	}

	// Кодируем и сохраняем изображение в формате JPG с указанным качеством
	jpegOptions := jpeg.Options{Quality: *quality}
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
	fmt.Printf("Размер входного файла: %d байт\n", inputFileSize)
	fmt.Printf("Размер выходного файла: %d байт\n", outputFileSize)
	fmt.Printf("Сокращение размера: %.2f%%\n", float64(inputFileSize-outputFileSize)/float64(inputFileSize)*100)

	if *rewrite {
		fmt.Println("Изображение успешно уменьшено и перезаписано как", *inputPath)
	} else {
		fmt.Println("Изображение успешно уменьшено и сохранено как", *outputPath)
	}
}

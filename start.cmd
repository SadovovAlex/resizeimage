resizeimage.exe -input "c:\Downloads\1111\222" -width 1600 -r -quality 70 -threads 8

pause

Использование:
  resize_image -input <input_dir> -width <width> [-r] [-quality <1-100>] [-threads <num>]
Параметры:
  -input   Путь к директории с изображениями (обязательный)
  -width   Новая ширина изображений (обязательный)
  -r       Перезаписать входные файлы (если указано)
  -quality Уровень качества выходных изображений (по умолчанию 100)
  -threads Количество параллельных потоков (по умолчанию 1)
  -help    Показать это сообщение
# ResizeImage

ResizeImage is a command-line tool designed for batch resizing of images from your phone or other devices. It utilizes the Lanczos interpolation method (a=3) for high-quality image resizing.

## Features

- **Batch Processing**: Resize multiple images in a single command.
- **Custom Width**: Specify a new width for the images (mandatory).
- **Quality Control**: Set the output image quality on a scale from 1 to 100.
- **Parallel Processing**: Adjust the number of threads for faster processing.
- **File Management**: Options to overwrite original files and set the current date for the output files.

## Usage

To use ResizeImage, you need to provide the following flags:

- `-input`: Path to the directory containing the images (required).
- `-width`: New width for the images (required).
- `-r`: Flag to overwrite the original files (optional).
- `-newdate`: Flag to set the current date for the output files (optional).
- `-quality`: Level of quality for the output images (default is 100).
- `-threads`: Number of parallel threads to use (default is 2).

### Example Command

```bash
go run main.go -input /path/to/images -width 1980 -quality 70 -threads 4
```

## Demo

For a demonstration of how the tool works, check out the following video: [Demo Video](https://github.com/SadovovAlex/resizeimage/blob/main/demo.mp4)

## Installation

To install ResizeImage, clone the repository and build the project:

```bash
git clone https://github.com/SadovovAlex/resizeimage.git
cd resizeimage
go build
```

## Release
get Release binary for your OS here [Release](https://github.com/SadovovAlex/resizeimage/blob/main/demo.mp4)

## Contributing

Contributions are welcome! If you have suggestions for improvements or new features, feel free to open an issue or submit a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
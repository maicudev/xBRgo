# xBRgo

**xBRgo** is a Go implementation of the xBR pixel-art scaling algorithm. It provides an importable Go package for integration into other projects and a command-line tool for batch image processing.

![](input/image.png)
![](output/image.png)

## Requirements

- Go 1.21 or newer

## Use as a Go package

Import the module and call `xbr.Scale` with any `image.Image`:

```go
package main

import (
	"image"
	"image/png"
	"os"

	xbr "github.com/maicudev/xBRgo"
)

func main() {
	input, err := os.Open("input.png")
	if err != nil {
		panic(err)
	}
	defer input.Close()

	src, _, err := image.Decode(input)
	if err != nil {
		panic(err)
	}

	scaled, err := xbr.Scale(src, 4, xbr.DefaultOptions())
	if err != nil {
		panic(err)
	}

	output, err := os.Create("output.png")
	if err != nil {
		panic(err)
	}
	defer output.Close()

	if err := png.Encode(output, scaled); err != nil {
		panic(err)
	}
}
```

Supported scale factors are `1`, `2`, `3`, `4`, `6`, and `8`. Factors `6` and
`8` use multiple xBR passes:

- `6x`: `3x` followed by `2x`
- `8x`: `4x` followed by `2x`

`xbr.DefaultOptions()` enables color blending. For custom behavior, pass an
`xbr.Options` value directly:

```go
options := xbr.Options{
	BlendColors: true,
	ScaleAlpha:  true,
}
scaled, err := xbr.Scale(src, 2, options)
```

## Command-line tool

Build the executable from the `cli` directory:

```bash
cd ./cli
go build -o ../xbr 
```

Run it with an input directory and an output directory:

```bash
cd ..
./xbr -factor 4 ./input ./output
```

The `-factor` option is optional and defaults to `2`. The tool processes PNG,
JPG, JPEG, and GIF files located directly inside the input directory. It does
not scan subdirectories. Results are always saved as PNG files in the output
directory, preserving each input file's base name. The output directory is
created automatically when it does not exist.

## Credits

This implementation is based on the original [xBRjs](https://github.com/joseprio/xBRjs) implementation and the xBR pixel-art scaling algorithm. The 2x, 3x, and 4x kernels, as well as the multi-pass behavior for 6x and 8x scaling, were ported to Go.

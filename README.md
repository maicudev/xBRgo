# xBRgo

This command-line tool scales pixel-art images using the xBR algorithm and
writes the results as PNG files.

![](input/image.png)
![](output/image.png)

## Requirements

- Go 1.21 or newer

## Usage

Run the program with an input directory and an output directory:

```bash
go run . [options] <input-directory> <output-directory>
```

Example:

```bash
go run . -factor 2 ./input ./output
```

The `-factor` option is optional and defaults to `2`. Supported values are
`2`, `3`, `4`, `6`, and `8`.

Factors `6` and `8` are implemented as multiple xBR passes:

- `6x`: `3x` followed by `2x`
- `8x`: `4x` followed by `2x`

The tool processes PNG, JPG, JPEG, and GIF files located directly inside the
input directory. Subdirectories are not scanned. Every output file is saved as
PNG in the output directory, preserving the input file's base name.

## Build and run the executable

Build the binary:

```bash
go build -o xbr .
```

Run it:

```bash
./xbr -factor 2 ./input ./output
```

## Credits

This implementation is based on the original [xBRjs](https://github.com/joseprio/xBRjs) implementation and the xBR pixel-art scaling algorithm. The 2x, 3x, and 4x kernels, as well as the multi-pass behavior for 6x and 8x scaling, were ported to Go.
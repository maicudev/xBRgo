# xBRgo

This command-line tool scales pixel-art images using the xBR algorithm and
writes the results as PNG files.

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

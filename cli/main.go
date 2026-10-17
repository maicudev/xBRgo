package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	xbr "github.com/maicudev/xBRgo"
)

func main() {
	factor := flag.Int("factor", 2, "scale factor: 2, 3, 4, 6, or 8")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <input-directory> <output-directory>\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	if !validFactor(*factor) {
		fmt.Fprintln(os.Stderr, "error: -factor must be 2, 3, 4, 6, or 8")
		os.Exit(2)
	}

	inputDir, outputDir := flag.Arg(0), flag.Arg(1)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fatalf("could not create output directory: %v", err)
	}

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		fatalf("could not read input directory: %v", err)
	}

	processed := 0
	for _, entry := range entries {
		if entry.IsDir() || !isImageFile(entry.Name()) {
			continue
		}

		inputPath := filepath.Join(inputDir, entry.Name())
		outputName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())) + ".png"
		outputPath := filepath.Join(outputDir, outputName)

		if err := convertFile(inputPath, outputPath, *factor); err != nil {
			fatalf("failed to convert %q: %v", inputPath, err)
		}
		processed++
		fmt.Printf("%s -> %s\n", inputPath, outputPath)
	}

	fmt.Printf("Converted %d image(s) with a %dx scale factor.\n", processed, *factor)
}

func validFactor(factor int) bool {
	return factor == 2 || factor == 3 || factor == 4 || factor == 6 || factor == 8
}

func isImageFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	default:
		return false
	}
}

func convertFile(inputPath, outputPath string, factor int) error {
	input, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer input.Close()

	src, _, err := image.Decode(input)
	if err != nil {
		return err
	}

	scaled, err := xbr.Scale(src, factor, xbr.DefaultOptions())
	if err != nil {
		return err
	}

	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()
	return png.Encode(output, scaled)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

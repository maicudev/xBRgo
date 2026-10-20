package xBRgo

import (
	"image"
	"image/color"
	"testing"
)

func testPixel(r, g, b, a uint8) pixel {
	return makePixel(color.NRGBA{R: r, G: g, B: b, A: a})
}

func TestTransparentPixelsDoNotCreateColorEdges(t *testing.T) {
	transparentRed := testPixel(255, 0, 0, 0)
	transparentBlue := testPixel(0, 0, 255, 0)
	opaqueBlue := testPixel(0, 0, 255, 255)

	if got := yuvDifference(transparentRed, transparentBlue, false); got != 0 {
		t.Fatalf("transparent RGB should be ignored, got difference %v", got)
	}
	if !isEqual(transparentRed, transparentBlue, false) {
		t.Fatal("transparent pixels with different hidden RGB should compare equal")
	}
	if isEqual(transparentRed, opaqueBlue, false) {
		t.Fatal("transparent and opaque pixels should not compare equal")
	}
}

func TestIsEqualUsesYUVThresholds(t *testing.T) {
	nearBlack := testPixel(100, 100, 100, 255)
	nearBlackWithDifferentRGB := testPixel(105, 105, 105, 255)
	white := testPixel(255, 255, 255, 255)

	if nearBlack == nearBlackWithDifferentRGB {
		t.Fatal("test pixels must have different packed RGB values")
	}
	if !isEqual(nearBlack, nearBlackWithDifferentRGB, false) {
		t.Fatal("nearby opaque colors should compare equal in YUV space")
	}
	if isEqual(nearBlack, white, false) {
		t.Fatal("visibly different opaque colors should not compare equal")
	}
}

func TestPixelInterpolateDoesNotBlendTransparentRGB(t *testing.T) {
	got := toNRGBAColor(pixelInterpolate(testPixel(255, 0, 0, 0), testPixel(0, 0, 255, 255), 1, 1))
	want := color.NRGBA{R: 0, G: 0, B: 255, A: 127}
	if got != want {
		t.Fatalf("transparent RGB leaked into blend: got %#v, want %#v", got, want)
	}
}

func TestScaleIsIndependentOfHiddenTransparentRGB(t *testing.T) {
	makeDiagonal := func(hidden color.NRGBA) *image.NRGBA {
		img := image.NewNRGBA(image.Rect(0, 0, 5, 5))
		for y := 0; y < 5; y++ {
			for x := 0; x < 5; x++ {
				img.SetNRGBA(x, y, hidden)
			}
		}
		for i := 0; i < 5; i++ {
			img.SetNRGBA(i, i, color.NRGBA{R: 40, G: 200, B: 120, A: 255})
		}
		return img
	}

	redHidden, _ := Scale(makeDiagonal(color.NRGBA{R: 255, A: 0}), 3, DefaultOptions())
	blueHidden, _ := Scale(makeDiagonal(color.NRGBA{B: 255, A: 0}), 3, DefaultOptions())
	if redHidden.Bounds() != blueHidden.Bounds() {
		t.Fatalf("scaled bounds differ: %v vs %v", redHidden.Bounds(), blueHidden.Bounds())
	}
	for y := redHidden.Bounds().Min.Y; y < redHidden.Bounds().Max.Y; y++ {
		for x := redHidden.Bounds().Min.X; x < redHidden.Bounds().Max.X; x++ {
			gotR, gotG, gotB, gotA := redHidden.At(x, y).RGBA()
			wantR, wantG, wantB, wantA := blueHidden.At(x, y).RGBA()
			if gotA != wantA || (gotA != 0 && (gotR != wantR || gotG != wantG || gotB != wantB)) {
				t.Fatalf("hidden transparent RGB changed visible pixel at (%d,%d): got %v, want %v", x, y, redHidden.At(x, y), blueHidden.At(x, y))
			}
		}
	}
}

func TestNoBlendPreservesExactPixelBlocks(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 200})
	src.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 128})
	src.SetNRGBA(1, 1, color.NRGBA{R: 12, G: 34, B: 56, A: 0})

	scaled, err := Scale(src, 3, Options{BlendColors: false})
	if err != nil {
		t.Fatal(err)
	}
	dst := scaled.(*image.NRGBA)
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			got := dst.NRGBAAt(x, y)
			want := src.NRGBAAt(x/3, y/3)
			if got != want {
				t.Fatalf("no-blend scaling changed pixel at (%d,%d): got %#v, want %#v", x, y, got, want)
			}
		}
	}
}

package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

type Options struct {
	BlendColors bool
	ScaleAlpha  bool
}

type pixel uint32

func Scale(src image.Image, factor int, options Options) (image.Image, error) {
	if factor < 1 {
		return nil, fmt.Errorf("invalid scale factor: %d", factor)
	}
	if factor == 1 {
		return toNRGBA(src), nil
	}

	passes := []int{factor}
	switch factor {
	case 6:
		passes = []int{3, 2}
	case 8:
		passes = []int{4, 2}
	}

	result := toNRGBA(src)
	for _, pass := range passes {
		if pass != 2 && pass != 3 && pass != 4 {
			return nil, fmt.Errorf("unsupported pass factor: %d", pass)
		}
		result = scalePass(result, pass, options)
	}
	return result, nil
}

func toNRGBA(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			dst.Set(x, y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}

func scalePass(src *image.NRGBA, factor int, options Options) *image.NRGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, sw*factor, sh*factor))
	srcPixels := make([]pixel, sw*sh)
	dstPixels := make([]pixel, sw*sh*factor*factor)

	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			srcPixels[x+y*sw] = makePixel(src.NRGBAAt(x, y))
		}
	}

	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			dx, dy := x*factor, y*factor
			switch factor {
			case 2:
				compute2x(srcPixels, x, y, sw, sh, dstPixels, dx, dy, sw*factor, options)
			case 3:
				compute3x(srcPixels, x, y, sw, sh, dstPixels, dx, dy, sw*factor, options)
			case 4:
				compute4x(srcPixels, x, y, sw, sh, dstPixels, dx, dy, sw*factor, options)
			}
		}
	}

	for y := 0; y < sh*factor; y++ {
		for x := 0; x < sw*factor; x++ {
			dst.SetNRGBA(x, y, toNRGBAColor(dstPixels[x+y*sw*factor]))
		}
	}
	return dst
}

func makePixel(c color.NRGBA) pixel {
	return pixel(uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A))
}

func toNRGBAColor(p pixel) color.NRGBA {
	return color.NRGBA{R: uint8(p >> 24), G: uint8(p >> 16), B: uint8(p >> 8), A: uint8(p)}
}

func getRelatedPoints(src []pixel, x, y, width, height int) [21]pixel {
	coords := [21][2]int{
		{x - 1, y - 2}, {x, y - 2}, {x + 1, y - 2},
		{x - 2, y - 1}, {x - 1, y - 1}, {x, y - 1}, {x + 1, y - 1}, {x + 2, y - 1},
		{x - 2, y}, {x - 1, y}, {x, y}, {x + 1, y}, {x + 2, y},
		{x - 2, y + 1}, {x - 1, y + 1}, {x, y + 1}, {x + 1, y + 1}, {x + 2, y + 1},
		{x - 1, y + 2}, {x, y + 2}, {x + 1, y + 2},
	}
	var result [21]pixel
	for i, point := range coords {
		px := point[0]
		py := point[1]
		if px < 0 {
			px = 0
		} else if px >= width {
			px = width - 1
		}
		if py < 0 {
			py = 0
		} else if py >= height {
			py = height - 1
		}
		result[i] = src[px+py*width]
	}
	return result
}

func rgba(p pixel) (float64, float64, float64, float64) {
	return float64(uint8(p >> 24)), float64(uint8(p >> 16)), float64(uint8(p >> 8)), float64(uint8(p))
}

func yuvDifference(a, b pixel, scaleAlpha bool) float64 {
	ar, ag, ab, aa := rgba(a)
	br, bg, bb, ba := rgba(b)
	ay := 0.299*ar + 0.587*ag + 0.114*ab
	au := -0.169*ar - 0.331*ag + 0.500*ab
	av := 0.500*ar - 0.419*ag - 0.081*ab
	by := 0.299*br + 0.587*bg + 0.114*bb
	bu := -0.169*br - 0.331*bg + 0.500*bb
	bv := 0.500*br - 0.419*bg - 0.081*bb
	diff := math.Abs(ay-by)*48 + math.Abs(au-bu)*7 + math.Abs(av-bv)*6
	if scaleAlpha {
		diff += math.Abs(aa-ba) * 48
	}
	return diff
}

func isEqual(a, b pixel, scaleAlpha bool) bool {
	if scaleAlpha {
		return a == b
	}
	return a>>8 == b>>8
}

func pixelInterpolate(dst, src pixel, dstWeight, srcWeight int) pixel {
	total := dstWeight + srcWeight
	dr, dg, db, da := rgba(dst)
	sr, sg, sb, sa := rgba(src)
	blend := func(a, b float64) uint8 {
		return uint8(math.Round((a*float64(dstWeight) + b*float64(srcWeight)) / float64(total)))
	}
	return makePixel(color.NRGBA{R: blend(dr, sr), G: blend(dg, sg), B: blend(db, sb), A: blend(da, sa)})
}

func blend32(dst, src pixel, blend bool) pixel {
	if blend {
		return pixelInterpolate(dst, src, 7, 1)
	}
	return dst
}
func blend64(dst, src pixel, blend bool) pixel {
	if blend {
		return pixelInterpolate(dst, src, 3, 1)
	}
	return dst
}
func blend128(dst, src pixel, blend bool) pixel {
	if blend {
		return pixelInterpolate(dst, src, 1, 1)
	}
	return dst
}
func blend192(dst, src pixel, blend bool) pixel {
	if blend {
		return pixelInterpolate(dst, src, 1, 3)
	}
	return src
}
func blend224(dst, src pixel, blend bool) pixel {
	if blend {
		return pixelInterpolate(dst, src, 1, 7)
	}
	return src
}

func kernel2(pe, pi, ph, pf, pg, pc, pd, pb, f4, i4, h5, i5, n1, n2, n3 pixel, options Options) [3]pixel {
	if pe != ph && pe != pf {
		e := yuvDifference(pe, pc, options.ScaleAlpha) + yuvDifference(pe, pg, options.ScaleAlpha) + yuvDifference(pi, h5, options.ScaleAlpha) + yuvDifference(pi, f4, options.ScaleAlpha) + 4*yuvDifference(ph, pf, options.ScaleAlpha)
		i := yuvDifference(ph, pd, options.ScaleAlpha) + yuvDifference(ph, i5, options.ScaleAlpha) + yuvDifference(pf, i4, options.ScaleAlpha) + yuvDifference(pf, pb, options.ScaleAlpha) + 4*yuvDifference(pe, pi, options.ScaleAlpha)
		px := pf
		if yuvDifference(pe, pf, options.ScaleAlpha) > yuvDifference(pe, ph, options.ScaleAlpha) {
			px = ph
		}
		if e < i && ((!isEqual(pf, pb, options.ScaleAlpha) && !isEqual(ph, pd, options.ScaleAlpha)) || (isEqual(pe, pi, options.ScaleAlpha) && !isEqual(pf, i4, options.ScaleAlpha) && !isEqual(ph, i5, options.ScaleAlpha)) || isEqual(pe, pg, options.ScaleAlpha) || isEqual(pe, pc, options.ScaleAlpha)) {
			ke := yuvDifference(pf, pg, options.ScaleAlpha)
			ki := yuvDifference(ph, pc, options.ScaleAlpha)
			ex2 := pe != pc && pb != pc
			ex3 := pe != pg && pd != pg
			if 2*ke <= ki && ex3 {
				n3 = blend192(n3, px, options.BlendColors)
				n2 = blend64(n2, px, options.BlendColors)
			}
			if ke >= 2*ki && ex2 {
				n3 = blend192(n3, px, options.BlendColors)
				n1 = blend64(n1, px, options.BlendColors)
			}
			if !(2*ke <= ki && ex3) && !(ke >= 2*ki && ex2) {
				n3 = blend128(n3, px, options.BlendColors)
			}
		} else if e <= i {
			n3 = blend64(n3, px, options.BlendColors)
		}
	}
	return [3]pixel{n1, n2, n3}
}

func compute2x(src []pixel, x, y, width, height int, dst []pixel, dx, dy, dstWidth int, options Options) {
	rp := getRelatedPoints(src, x, y, width, height)
	a1, b1, c1, a0, pa, pb, pc, c4, d0, pd, pe, pf, f4, g0, pg, ph, pi, i4, g5, h5, i5 := rp[0], rp[1], rp[2], rp[3], rp[4], rp[5], rp[6], rp[7], rp[8], rp[9], rp[10], rp[11], rp[12], rp[13], rp[14], rp[15], rp[16], rp[17], rp[18], rp[19], rp[20]
	e0, e1, e2, e3 := pe, pe, pe, pe
	r := kernel2(pe, pi, ph, pf, pg, pc, pd, pb, f4, i4, h5, i5, e1, e2, e3, options)
	e1, e2, e3 = r[0], r[1], r[2]
	r = kernel2(pe, pc, pf, pb, pi, pa, ph, pd, b1, c1, f4, c4, e0, e3, e1, options)
	e0, e3, e1 = r[0], r[1], r[2]
	r = kernel2(pe, pa, pb, pd, pc, pg, pf, ph, d0, a0, b1, a1, e2, e1, e0, options)
	e2, e1, e0 = r[0], r[1], r[2]
	r = kernel2(pe, pg, pd, ph, pa, pi, pb, pf, h5, g5, d0, g0, e3, e0, e2, options)
	e3, e0, e2 = r[0], r[1], r[2]
	dst[dx+dy*dstWidth], dst[dx+1+dy*dstWidth], dst[dx+(dy+1)*dstWidth], dst[dx+1+(dy+1)*dstWidth] = e0, e1, e2, e3
}

func kernel3(pe, pi, ph, pf, pg, pc, pd, pb, f4, i4, h5, i5, n2, n5, n6, n7, n8 pixel, options Options) [5]pixel {
	if pe != ph && pe != pf {
		e := yuvDifference(pe, pc, options.ScaleAlpha) + yuvDifference(pe, pg, options.ScaleAlpha) + yuvDifference(pi, h5, options.ScaleAlpha) + yuvDifference(pi, f4, options.ScaleAlpha) + 4*yuvDifference(ph, pf, options.ScaleAlpha)
		i := yuvDifference(ph, pd, options.ScaleAlpha) + yuvDifference(ph, i5, options.ScaleAlpha) + yuvDifference(pf, i4, options.ScaleAlpha) + yuvDifference(pf, pb, options.ScaleAlpha) + 4*yuvDifference(pe, pi, options.ScaleAlpha)
		state := e < i && ((!isEqual(pf, pb, options.ScaleAlpha) && !isEqual(pf, pc, options.ScaleAlpha)) || (!isEqual(ph, pd, options.ScaleAlpha) && !isEqual(ph, pg, options.ScaleAlpha)) || (isEqual(pe, pi, options.ScaleAlpha) && ((!isEqual(pf, f4, options.ScaleAlpha) && !isEqual(pf, i4, options.ScaleAlpha)) || (!isEqual(ph, h5, options.ScaleAlpha) && !isEqual(ph, i5, options.ScaleAlpha)))) || isEqual(pe, pg, options.ScaleAlpha) || isEqual(pe, pc, options.ScaleAlpha))
		px := pf
		if yuvDifference(pe, pf, options.ScaleAlpha) > yuvDifference(pe, ph, options.ScaleAlpha) {
			px = ph
		}
		if state {
			ke, ki := yuvDifference(pf, pg, options.ScaleAlpha), yuvDifference(ph, pc, options.ScaleAlpha)
			ex2, ex3 := pe != pc && pb != pc, pe != pg && pd != pg
			if 2*ke <= ki && ex3 && ke >= 2*ki && ex2 {
				bl7 := blend192(n7, px, options.BlendColors)
				n7 = bl7
				n5 = bl7
				bl6 := blend64(n6, px, options.BlendColors)
				n6 = bl6
				n2 = bl6
				n8 = px
			} else if 2*ke <= ki && ex3 {
				n7 = blend192(n7, px, options.BlendColors)
				n5 = blend64(n5, px, options.BlendColors)
				n6 = blend64(n6, px, options.BlendColors)
				n8 = px
			} else if ke >= 2*ki && ex2 {
				n5 = blend192(n5, px, options.BlendColors)
				n7 = blend64(n7, px, options.BlendColors)
				n2 = blend64(n2, px, options.BlendColors)
				n8 = px
			} else {
				n8 = blend224(n8, px, options.BlendColors)
				n5 = blend32(n5, px, options.BlendColors)
				n7 = blend32(n7, px, options.BlendColors)
			}
		} else if e <= i {
			n8 = blend128(n8, px, options.BlendColors)
		}
	}
	return [5]pixel{n2, n5, n6, n7, n8}
}

func compute3x(src []pixel, x, y, width, height int, dst []pixel, dx, dy, dstWidth int, options Options) {
	rp := getRelatedPoints(src, x, y, width, height)
	a1, b1, c1, a0, pa, pb, pc, c4, d0, pd, pe, pf, f4, g0, pg, ph, pi, i4, g5, h5, i5 := rp[0], rp[1], rp[2], rp[3], rp[4], rp[5], rp[6], rp[7], rp[8], rp[9], rp[10], rp[11], rp[12], rp[13], rp[14], rp[15], rp[16], rp[17], rp[18], rp[19], rp[20]
	e0, e1, e2, e3, e4, e5, e6, e7, e8 := pe, pe, pe, pe, pe, pe, pe, pe, pe
	r := kernel3(pe, pi, ph, pf, pg, pc, pd, pb, f4, i4, h5, i5, e2, e5, e6, e7, e8, options)
	e2, e5, e6, e7, e8 = r[0], r[1], r[2], r[3], r[4]
	r = kernel3(pe, pc, pf, pb, pi, pa, ph, pd, b1, c1, f4, c4, e0, e1, e8, e5, e2, options)
	e0, e1, e8, e5, e2 = r[0], r[1], r[2], r[3], r[4]
	r = kernel3(pe, pa, pb, pd, pc, pg, pf, ph, d0, a0, b1, a1, e6, e3, e2, e1, e0, options)
	e6, e3, e2, e1, e0 = r[0], r[1], r[2], r[3], r[4]
	r = kernel3(pe, pg, pd, ph, pa, pi, pb, pf, h5, g5, d0, g0, e8, e7, e0, e3, e6, options)
	e8, e7, e0, e3, e6 = r[0], r[1], r[2], r[3], r[4]
	vals := [9]pixel{e0, e1, e2, e3, e4, e5, e6, e7, e8}
	for oy := 0; oy < 3; oy++ {
		for ox := 0; ox < 3; ox++ {
			dst[dx+ox+(dy+oy)*dstWidth] = vals[ox+oy*3]
		}
	}
}

func kernel4(pe, pi, ph, pf, pg, pc, pd, pb, f4, i4, h5, i5, n15, n14, n11, n3, n7, n10, n13, n12 pixel, options Options) [8]pixel {
	if pe != ph && pe != pf {
		e := yuvDifference(pe, pc, options.ScaleAlpha) + yuvDifference(pe, pg, options.ScaleAlpha) + yuvDifference(pi, h5, options.ScaleAlpha) + yuvDifference(pi, f4, options.ScaleAlpha) + 4*yuvDifference(ph, pf, options.ScaleAlpha)
		i := yuvDifference(ph, pd, options.ScaleAlpha) + yuvDifference(ph, i5, options.ScaleAlpha) + yuvDifference(pf, i4, options.ScaleAlpha) + yuvDifference(pf, pb, options.ScaleAlpha) + 4*yuvDifference(pe, pi, options.ScaleAlpha)
		px := pf
		if yuvDifference(pe, pf, options.ScaleAlpha) > yuvDifference(pe, ph, options.ScaleAlpha) {
			px = ph
		}
		if e < i && ((!isEqual(pf, pb, options.ScaleAlpha) && !isEqual(ph, pd, options.ScaleAlpha)) || (isEqual(pe, pi, options.ScaleAlpha) && !isEqual(pf, i4, options.ScaleAlpha) && !isEqual(ph, i5, options.ScaleAlpha)) || isEqual(pe, pg, options.ScaleAlpha) || isEqual(pe, pc, options.ScaleAlpha)) {
			ke, ki := yuvDifference(pf, pg, options.ScaleAlpha), yuvDifference(ph, pc, options.ScaleAlpha)
			ex2, ex3 := pe != pc && pb != pc, pe != pg && pd != pg
			if 2*ke <= ki && ex3 {
				n15 = px
				n14 = px
				n11 = blend192(n11, px, options.BlendColors)
				n13 = blend192(n13, px, options.BlendColors)
				n12 = blend64(n12, px, options.BlendColors)
				n10 = blend64(n10, px, options.BlendColors)
			}
			if ke >= 2*ki && ex2 {
				n15 = px
				n14 = blend192(n14, px, options.BlendColors)
				n11 = px
				n3 = blend64(n3, px, options.BlendColors)
				n7 = blend192(n7, px, options.BlendColors)
				n10 = blend64(n10, px, options.BlendColors)
			}
			if !(2*ke <= ki && ex3) && !(ke >= 2*ki && ex2) {
				n15 = px
				n14 = blend128(n14, px, options.BlendColors)
				n11 = blend128(n11, px, options.BlendColors)
			}
		} else if e <= i {
			n15 = blend128(n15, px, options.BlendColors)
		}
	}
	return [8]pixel{n15, n14, n11, n3, n7, n10, n13, n12}
}

func compute4x(src []pixel, x, y, width, height int, dst []pixel, dx, dy, dstWidth int, options Options) {
	rp := getRelatedPoints(src, x, y, width, height)
	a1, b1, c1, a0, pa, pb, pc, c4, d0, pd, pe, pf, f4, g0, pg, ph, pi, i4, g5, h5, i5 := rp[0], rp[1], rp[2], rp[3], rp[4], rp[5], rp[6], rp[7], rp[8], rp[9], rp[10], rp[11], rp[12], rp[13], rp[14], rp[15], rp[16], rp[17], rp[18], rp[19], rp[20]
	e0, e1, e2, e3, e4, e5, e6, e7, e8, e9, ea, eb, ec, ed, ee, ef := pe, pe, pe, pe, pe, pe, pe, pe, pe, pe, pe, pe, pe, pe, pe, pe
	r := kernel4(pe, pi, ph, pf, pg, pc, pd, pb, f4, i4, h5, i5, ef, ee, eb, e3, e7, ea, ed, ec, options)
	ef, ee, eb, e3, e7, ea, ed, ec = r[0], r[1], r[2], r[3], r[4], r[5], r[6], r[7]
	r = kernel4(pe, pc, pf, pb, pi, pa, ph, pd, b1, c1, f4, c4, e3, e7, e2, e0, e1, e6, eb, ef, options)
	e3, e7, e2, e0, e1, e6, eb, ef = r[0], r[1], r[2], r[3], r[4], r[5], r[6], r[7]
	r = kernel4(pe, pa, pb, pd, pc, pg, pf, ph, d0, a0, b1, a1, e0, e1, e4, ec, e8, e5, e2, e3, options)
	e0, e1, e4, ec, e8, e5, e2, e3 = r[0], r[1], r[2], r[3], r[4], r[5], r[6], r[7]
	r = kernel4(pe, pg, pd, ph, pa, pi, pb, pf, h5, g5, d0, g0, ec, e8, ed, ef, ee, e9, e4, e0, options)
	ec, e8, ed, ef, ee, e9, e4, e0 = r[0], r[1], r[2], r[3], r[4], r[5], r[6], r[7]
	vals := [16]pixel{e0, e1, e2, e3, e4, e5, e6, e7, e8, e9, ea, eb, ec, ed, ee, ef}
	for oy := 0; oy < 4; oy++ {
		for ox := 0; ox < 4; ox++ {
			dst[dx+ox+(dy+oy)*dstWidth] = vals[ox+oy*4]
		}
	}
}

//
// main.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"os"

	"github.com/fogleman/gg"
)

var (
	white     = color.RGBA{255, 255, 255, 255}
	black     = color.RGBA{0, 0, 0, 255}
	red       = color.RGBA{255, 0, 0, 255}
	darkCyan  = color.RGBA{50, 144, 172, 255}
	measureDC = gg.NewContext(512, 512)
	font      = "/Users/mrossi/Downloads/computer-modern/cmuntb.ttf"
	fontSize  = float64(16)
)

func main() {
	flag.Parse()

	err := measureDC.LoadFontFace(font, fontSize)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range flag.Args() {
		err := createPNG(f)
		if err != nil {
			log.Fatal("Failed to process file '%s': %s\n", f, err)
		}
	}
}

func createPNG(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	thumb := fmt.Sprintf("%x\u2026%x", sum[0:4], sum[28:32])

	title := "PKIback.com"
	titleW, titleH := measureDC.MeasureString(title)
	thumbW, thumbH := measureDC.MeasureString(thumb)

	certWidth := int(titleW)
	if int(thumbW) > certWidth {
		certWidth = int(thumbW)
	}

	pixels := len(data)/4 + 1
	fmt.Printf("#pixels=%d\n", pixels)

	certHeight := pixels / certWidth
	if (pixels % certWidth) != 0 {
		certHeight++
	}

	marginX := 4
	marginY := 4

	imageWidth := certWidth + marginX*2
	imageHeight := int(titleH) + certHeight + int(thumbH) + 4*marginY

	dc := gg.NewContext(imageWidth, imageHeight)
	dc.SetRGB255(89, 211, 243)
	dc.Clear()
	dc.SetRGB255(0, 0, 0)
	err = dc.LoadFontFace(font, fontSize)
	if err != nil {
		log.Fatal(err)
	}
	dc.DrawStringAnchored(title,
		float64(imageWidth/2), float64(marginY)+titleH/2,
		0.5, 0.5)
	dc.DrawStringAnchored(thumb, float64(imageWidth/2),
		float64(imageHeight)-thumbH/2-float64(marginY), 0.5, 0.5)

	buf := new(bytes.Buffer)

	err = dc.EncodePNG(buf)
	if err != nil {
		log.Fatal(err)
	}
	img, _, err := image.Decode(buf)
	if err != nil {
		log.Fatal(err)
	}
	rgba := img.(*image.RGBA)

	certX := marginX
	certY := imageHeight/2 - certHeight/2 + 1

	var dataPos int
	for y := 0; y < certHeight; y++ {
		for x := 0; x < certWidth; x++ {
			var pixel uint32
			if y == 0 && x == 0 {
				pixel = uint32(len(data))
			} else {
				for i := 0; i < 4; i++ {
					if dataPos >= len(data) {
						break
					}
					pixel <<= 8
					pixel += uint32(data[dataPos])
					dataPos++
				}
			}
			color := color.RGBA{
				R: uint8((pixel >> 24) & 0xff),
				G: uint8((pixel >> 16) & 0xff),
				B: uint8((pixel >> 8) & 0xff),
				A: uint8(pixel & 0xff),
			}
			rgba.SetRGBA(certX+x, certY+y, color)
		}
	}

	Stroke(rgba, rgba.Bounds(), darkCyan)

	f, err := os.Create(",png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	return png.Encode(f, rgba)
}

func Stroke(rgba *image.RGBA, r image.Rectangle, c color.RGBA) {
	for x := r.Min.X; x < r.Max.X; x++ {
		rgba.SetRGBA(x, 0, c)
		rgba.SetRGBA(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		rgba.SetRGBA(0, y, c)
		rgba.SetRGBA(r.Max.X-1, y, c)
	}
}

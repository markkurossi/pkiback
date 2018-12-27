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
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/freetype"
	"golang.org/x/image/font"
)

const (
	bytesPerPixel = 3
	headerLen     = 12
	headerPixels  = 4
	headerMagic   = 0x504b4901
)

type Theme struct {
	FG     color.RGBA
	Thumb  color.RGBA
	BG     color.RGBA
	Border color.RGBA
}

var (
	white    = color.RGBA{255, 255, 255, 255}
	black    = color.RGBA{0, 0, 0, 255}
	red      = color.RGBA{255, 0, 0, 255}
	cyan     = color.RGBA{89, 211, 243, 255}
	darkCyan = color.RGBA{50, 144, 172, 255}
	theme1   = Theme{
		FG:     black,
		Thumb:  black,
		BG:     color.RGBA{101, 190, 235, 255},
		Border: darkCyan,
	}
	theme2 = Theme{
		FG:     black,
		Thumb:  black,
		BG:     color.RGBA{131, 161, 168, 255},
		Border: color.RGBA{58, 93, 102, 255},
	}
	theme3 = Theme{
		FG:     color.RGBA{239, 240, 245, 255},
		Thumb:  color.RGBA{201, 79, 65, 255},
		BG:     color.RGBA{46, 48, 61, 255},
		Border: black,
	}
	theme4 = Theme{
		FG:     white,
		Thumb:  black,
		BG:     color.RGBA{65, 110, 159, 255},
		Border: color.RGBA{25, 72, 129, 255},
	}
	theme5 = Theme{
		FG:     color.RGBA{238, 238, 238, 255},
		Thumb:  black,
		BG:     color.RGBA{76, 107, 134, 255},
		Border: black,
	}
	fontName = "/Users/mrossi/Downloads/liberation-mono/LiberationMono-Regular.ttf"
	//fontName = "/Users/mrossi/Downloads/monospace-typewriter/MonospaceTypewriter.ttf"
	fontSize = float64(16)
	verbose  = false
)

func main() {
	verbosep := flag.Bool("verbose", false, "Verbose output.")
	encode := flag.String("encode", "",
		"Encode argument file into PKIback image.")
	decode := flag.String("decode", "", "Decode argument PKIback image.")
	flag.Parse()

	verbose = *verbosep

	if len(*encode) > 0 {
		err := EncodeImage(*encode, theme1)
		if err != nil {
			log.Fatalf("Failed to process file '%s': %s\n", *encode, err)
		}
	}
	if len(*decode) > 0 {
		err := DecodeImage(*decode)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func EncodeImage(path string, theme Theme) error {
	certData, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(certData)
	if verbose {
		fmt.Printf("SHA256: input:\n%sDigest:\n%s",
			hex.Dump(certData),
			hex.Dump(sum[:]))
	}

	thumb := fmt.Sprintf("%x\u2026%x", sum[0:4], sum[28:32])

	title := "PKIback.com"

	fontData, err := ioutil.ReadFile(fontName)
	if err != nil {
		return err
	}
	fnt, err := freetype.ParseFont(fontData)
	if err != nil {
		return err
	}

	ctx := freetype.NewContext()
	ctx.SetDPI(72)
	ctx.SetHinting(font.HintingFull)
	ctx.SetFont(fnt)
	ctx.SetFontSize(fontSize)

	thumbSize, err := ctx.DrawString(thumb, freetype.Pt(0, 0))
	if err != nil {
		return err
	}
	certWidth := thumbSize.X.Ceil()

	titleSize, err := ctx.DrawString(title, freetype.Pt(0, 0))
	if err != nil {
		return err
	}
	if titleSize.X.Ceil() > certWidth {
		certWidth = titleSize.X.Ceil()
	}

	fmt.Printf("certWidth: %d\n", certWidth)
	data := encodeData(certData, sum[:], certWidth)

	pixels := len(data) / bytesPerPixel
	if (len(data) % bytesPerPixel) != 0 {
		pixels++
	}
	fmt.Printf("#pixels=%d\n", pixels)

	certHeight := pixels / certWidth
	if (pixels % certWidth) != 0 {
		certHeight++
	}

	marginX := 6
	marginY := 6

	imageWidth := certWidth + marginX*2
	imageHeight := int(fontSize) + certHeight + int(fontSize) + 4*marginY

	rgba := image.NewRGBA(image.Rect(0, 0, imageWidth, imageHeight))
	draw.Draw(rgba, rgba.Bounds(), image.NewUniform(theme.BG), image.ZP,
		draw.Src)

	ctx.SetClip(rgba.Bounds())
	ctx.SetDst(rgba)
	ctx.SetSrc(image.NewUniform(theme.FG))

	pt := freetype.Pt(imageWidth, 0)
	pt = pt.Sub(titleSize)
	pt = pt.Div(freetype.Pt(2, 0).X)
	pt.Y = freetype.Pt(0, marginY+int(fontSize)-1).Y
	ctx.DrawString(title, pt)

	ctx.SetSrc(image.NewUniform(theme.Thumb))

	pt = freetype.Pt(imageWidth, 0)
	pt = pt.Sub(thumbSize)
	pt = pt.Div(freetype.Pt(2, 0).X)
	pt.Y = freetype.Pt(0, imageHeight-marginY-1).Y
	ctx.DrawString(thumb, pt)

	certX := marginX
	certY := imageHeight/2 - certHeight/2 + 1

	var dataPos int
	for y := 0; y < certHeight; y++ {
		for x := 0; x < certWidth; x++ {
			var pixel uint32
			for i := 0; i < bytesPerPixel; i++ {
				pixel <<= 8
				if dataPos < len(data) {
					pixel |= uint32(data[dataPos])
					dataPos++
				}
			}
			for i := bytesPerPixel; i < 4; i++ {
				pixel <<= 8
				pixel |= 0xff
			}
			rgba.SetRGBA(certX+x, certY+y, ValueToColor(pixel))
		}
	}

	if false {
		Stroke(rgba, image.Rectangle{
			Min: image.Point{
				X: certX - 1,
				Y: certY - 1,
			},
			Max: image.Point{
				X: certX + certWidth + 1,
				Y: certY + certHeight + 1,
			},
		}, theme.Border)
	}
	Stroke(rgba, rgba.Bounds(), theme.Border)

	f, err := os.Create(",png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	return png.Encode(f, rgba)
}

func ValueToColor(value uint32) color.RGBA {
	return color.RGBA{
		R: uint8((value >> 24) & 0xff),
		G: uint8((value >> 16) & 0xff),
		B: uint8((value >> 8) & 0xff),
		A: uint8(value & 0xff),
	}
}

func ValueBytes(value uint32) []byte {
	return []byte{
		byte((value >> 24) & 0xff),
		byte((value >> 16) & 0xff),
		byte((value >> 8) & 0xff),
	}
}

func ColorToValue(c color.RGBA) (value uint32) {
	value |= uint32(c.R)
	value <<= 8
	value |= uint32(c.G)
	value <<= 8
	value |= uint32(c.B)
	value <<= 8
	value |= uint32(c.A)
	return value
}

func encodeData(data, digest []byte, width int) []byte {

	bo := binary.BigEndian

	buf := new(bytes.Buffer)
	binary.Write(buf, bo, uint32(headerMagic))
	binary.Write(buf, bo, uint32(len(data)))
	binary.Write(buf, bo, uint32(width))
	buf.Write(data)
	buf.Write(digest)

	return buf.Bytes()
}

func decodeHeader(hdr []byte) (int, int, int) {
	return int(binary.BigEndian.Uint32(hdr[0:])),
		int(binary.BigEndian.Uint32(hdr[4:])),
		int(binary.BigEndian.Uint32(hdr[8:]))
}

func Stroke(rgba *image.RGBA, r image.Rectangle, c color.RGBA) {
	for x := r.Min.X; x < r.Max.X; x++ {
		rgba.SetRGBA(x, r.Min.Y, c)
		rgba.SetRGBA(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		rgba.SetRGBA(r.Min.X, y, c)
		rgba.SetRGBA(r.Max.X-1, y, c)
	}
}

func DecodeImage(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	rgba := img.(*image.RGBA)
	x, y, length, width := findCert(rgba)
	if length == 0 {
		return fmt.Errorf("No certificate found")
	}
	if verbose {
		fmt.Printf("Certificate found from %dx%d: len=%d, width=%d\n",
			x, y, length, width)
	}

	data, err = readData(rgba, x, y, headerLen+length+sha256.Size, width)
	if err != nil {
		return err
	}
	hdr := data[0:headerLen]
	certData := data[headerLen : headerLen+length]
	digest := data[headerLen+length:]

	if verbose {
		fmt.Printf("Header:\n%s", hex.Dump(hdr))
		fmt.Printf("Certificate:\n%s", hex.Dump(certData))
		fmt.Printf("Digest:\n%s", hex.Dump(digest))
	}

	computed := sha256.Sum256(certData)
	if bytes.Compare(computed[:], digest) != 0 {
		return fmt.Errorf("Digest mismatch")
	}

	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return err
	}

	fmt.Printf("Certificate:\nSubject: %s\n", cert.Subject)

	return nil
}

func findCert(rgba *image.RGBA) (x, y, length, width int) {
	bounds := rgba.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if bounds.Max.X-x >= headerPixels {
				var hdr []byte

				for i := 0; i < headerPixels; i++ {
					value := ColorToValue(rgba.RGBAAt(x+i, y))
					hdr = append(hdr, ValueBytes(value)...)
				}
				magic, length, width := decodeHeader(hdr)
				if magic == headerMagic && width <= bounds.Max.X-x {
					digestLen := sha256.Size
					if headerLen+length+digestLen <=
						(bounds.Max.Y-y)*width*bytesPerPixel {
						return x, y, length, width
					}
				}
			}
		}
	}
	return -1, -1, 0, 0
}

func readData(rgba *image.RGBA, sx, sy, length, width int) ([]byte, error) {
	bounds := rgba.Bounds()
	var result []byte

	for y := sy; y < bounds.Max.Y; y++ {
		for x := 0; x < width; x++ {
			value := ColorToValue(rgba.RGBAAt(x+sx, y))
			result = append(result, ValueBytes(value)...)
			if len(result) >= length {
				break
			}
		}
	}
	if len(result) < length {
		return nil, fmt.Errorf("Not enough data found: %d < %d",
			len(result), length)
	}
	return result[:length], nil
}

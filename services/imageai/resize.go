package imageai

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"

	"golang.org/x/image/draw"
)

const (
	ShortSideMax = 768
	LongSideMax  = 2000
	JpegQuality  = 85
)

// Resize 將圖片縮放為短邊 768px、長邊不超過 2000px，維持比例。
// 支援 JPEG、PNG 解碼；輸出為 JPEG（品質 85%）。
// 若原始尺寸已符合，直接縮放或回傳編碼結果。
func Resize(r io.Reader) ([]byte, string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, "", err
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	short, long := w, h
	if short > long {
		short, long = long, short
	}

	scale := 1.0
	if short > ShortSideMax {
		scale = float64(ShortSideMax) / float64(short)
	}
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)

	// 若長邊仍超過 2000，再縮放
	newShort, newLong := newW, newH
	if newShort > newLong {
		newShort, newLong = newLong, newShort
	}
	if newLong > LongSideMax {
		scale2 := float64(LongSideMax) / float64(newLong)
		newW = int(float64(newW) * scale2)
		newH = int(float64(newH) * scale2)
	}

	if newW <= 0 {
		newW = 1
	}
	if newH <= 0 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: JpegQuality}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "image/jpeg", nil
}

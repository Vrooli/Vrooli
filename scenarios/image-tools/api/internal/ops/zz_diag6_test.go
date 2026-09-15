package ops

import (
	"image/color"
	"os"
	"testing"
)

func TestZZDiag6(t *testing.T) {
	ref, _ := os.ReadFile("testdata/aquila-clean-512.png")
	refImg, _, _ := Decode(ref)
	b := refImg.Bounds()
	for _, p := range []struct{x,y int}{{256,80},{256,150},{256,256},{256,400},{256,440},{150,256},{350,256},{100,100},{410,410},{256,180}} {
		c := color.NRGBAModel.Convert(refImg.At(b.Min.X+p.x,b.Min.Y+p.y)).(color.NRGBA)
		t.Logf("ref(%d,%d)=%v", p.x,p.y,c)
	}
}

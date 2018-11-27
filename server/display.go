package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"log"
	"net"
	"os"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/bamiaux/rez"
	"github.com/pixiv/go-libjpeg/jpeg"
)

func CaptureScreen(c *xgb.Conn) (*image.RGBA, error) {
	screen := xproto.Setup(c).DefaultScreen(c)
	x := screen.WidthInPixels
	y := screen.HeightInPixels
	rect := image.Rect(0, 0, int(x), int(y))

	xImg, err := xproto.GetImage(c, xproto.ImageFormatZPixmap, xproto.Drawable(screen.Root), int16(rect.Min.X), int16(rect.Min.Y), uint16(x), uint16(y), 0xffffffff).Reply()
	if err != nil {
		return nil, err
	}

	data := xImg.Data

	for i := 0; i < len(data); i += 4 {
		data[i], data[i+2], data[i+3] = data[i+2], data[i], 255
	}

	img := &image.RGBA{data, 4 * int(x), image.Rect(0, 0, int(x), int(y))}
	return img, nil
}

const (
	CONN_HOST = "0.0.0.0"
	CONN_PORT = "5431"
	CONN_TYPE = "tcp"
)

func handleRequest(conn net.Conn) {
	defer conn.Close()

	resx := 1280
	resy := 720

	c, err := xgb.NewConn()
	if err != nil {
		log.Fatalf("Could not get xgb-conn: %v", err)
	}
	defer c.Close()

	screen := xproto.Setup(c).DefaultScreen(c)
	x := int(screen.WidthInPixels)
	y := int(screen.HeightInPixels)

	// TODO: We need to update the image-resolution if it changes
	dstFrame := image.NewRGBA(image.Rect(0, 0, resx, resy))
	converter, _ := rez.NewConverter(&rez.ConverterConfig{
		Input: rez.Descriptor{
			Width:      x,
			Height:     y,
			Interlaced: false,
			Ratio:      rez.Ratio444,
			Pack:       4,
			Planes:     1,
		},
		Output: rez.Descriptor{
			Width:      resx,
			Height:     resy,
			Interlaced: false,
			Ratio:      rez.Ratio444,
			Pack:       4,
			Planes:     1,
		},
	}, rez.NewBilinearFilter())

	j := 0
	for {
		framelimiter := time.NewTimer(time.Second / 60)

		//outimg := resize.Resize(resx, resy, img, resize.Lanczos2)
		img, err := CaptureScreen(c)
		if err != nil {
			panic(err)
		}

		j++
		if j%60 == 0 {
			fmt.Println(j)
		}

		buf := new(bytes.Buffer)

		if x != resx || y != resy {
			// Downscale if necessary
			converter.Convert(dstFrame, img)
			jpeg.Encode(buf, dstFrame, &jpeg.EncoderOptions{Quality: 15})
		} else {
			jpeg.Encode(buf, img, &jpeg.EncoderOptions{Quality: 15})
		}

		bs := make([]byte, 4)
		binary.LittleEndian.PutUint32(bs, uint32(buf.Len()))
		_, err = conn.Write(bs)
		if err != nil {
			break
		}
		_, err = conn.Write(buf.Bytes())
		if err != nil {
			break
		}

		<-framelimiter.C
	}

}

func main() {

	l, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}

}

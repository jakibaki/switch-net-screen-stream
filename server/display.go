package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"log"
	"net"
	"os"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
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
	// Make a buffer to hold incoming data.
	//buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	/*	reqLen, err := conn.Read(buf)
		if err != nil {
		  fmt.Println("Error reading:", err.Error())
		}
		// Send a response back to person contacting us.
		conn.Write([]byte("Message received."))
		// Close the connection when you're done with it.*/

	//resx := 1280
	//resy := 720

	c, err := xgb.NewConn()
	if err != nil {
		log.Fatalf("Could not get xgb-conn: %v", err)
	}
	defer c.Close()

	/*dstFrame := image.NewRGBA(image.Rect(0, 0, resx, resy))
	converter, _ := rez.NewConverter(&rez.ConverterConfig{
		Input: rez.Descriptor{
			Width:      1920,
			Height:     1080,
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
	}, rez.NewBilinearFilter())*/

	j := 0
	for {
		//outimg := resize.Resize(resx, resy, img, resize.Lanczos2)
		img, err := CaptureScreen(c)
		if err != nil {
			panic(err)
		}

		j++
		if j%60 == 0 {
			fmt.Println(j)
		}

		//converter.Convert(dstFrame, img)

		buf := new(bytes.Buffer)
		jpeg.Encode(buf, img, &jpeg.EncoderOptions{Quality: 30})

		bs := make([]byte, 4)
		binary.LittleEndian.PutUint32(bs, uint32(buf.Len()))
		conn.Write(bs)
		go conn.Write(buf.Bytes())
	}

	conn.Close()
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

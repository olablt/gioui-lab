package ui

import (
	"bytes"
	"image"
	"image/png"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/gpu/headless"
	"gioui.org/io/event"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func GetShaper() *text.Shaper {
	// load Awesome font
	fontData, err := os.ReadFile("assets/Font Awesome 5 Pro-Light-300.otf")
	if err != nil {
		log.Println("[ERROR] Error loading font file:", err)
		shaper := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
		return shaper
	}

	// 1. opentype parse
	// Parse the font
	face, err := opentype.Parse(fontData)
	if err != nil {
		log.Fatal(err)
	}
	// Create font collection
	fontAwesome := []font.FontFace{
		{
			Font: font.Font{
				Typeface: "FontAwesome",
			},
			Face: face,
		},
	}

	// // 2.0
	// fontAwesome, err := opentype.ParseCollection(fontData)
	// if err != nil {
	// 	panic(fmt.Errorf("failed to parse font: %v", err))
	// }

	// merge go font and awsome font
	faces := []font.FontFace{}
	faces = append(gofont.Collection(), fontAwesome...)
	for i, face := range faces {
		log.Printf("#%v face %+v", i, face)
	}

	shaper := text.NewShaper(text.NoSystemFonts(), text.WithCollection(faces))
	return shaper
}

func Loop(refresh chan struct{}, fn func(win *app.Window, gtx layout.Context, th *material.Theme), onDestory func()) {
	th := material.NewTheme()
	// th.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	th.Shaper = GetShaper()

	// Create signal channel for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		w := &app.Window{}
		w.Option(
			app.Title("oGio"),
			app.Size(unit.Dp(1920/2), unit.Dp(1080/2)),
		)

		// Make a channel to read window events from.
		events := make(chan event.Event)
		// Make a channel to signal the end of processing a window event.
		acks := make(chan struct{})

		// Create a done channel to signal shutdown
		done := make(chan struct{})
		go func() {
			// Handle OS signals
			select {
			case <-sigChan:
				log.Println("Received interrupt signal")
				onDestory()
				close(done)
				os.Exit(0)
			case <-done:
				return
			}
		}()

		go func() {
			// Iterate window events
			for {
				select {
				case <-done:
					return
				default:
					ev := w.Event()
					events <- ev
					<-acks
					if _, ok := ev.(app.DestroyEvent); ok {
						return
					}
				}
			}
		}()
		// var ed widget.Editor
		var ops op.Ops
		for {
			select {
			case <-done:
				return
			case event := <-events:
				switch event := event.(type) {
				case app.DestroyEvent:
					// We must manually ack a destroy event in order to ensure that the other goroutine
					// shuts down when we return.
					acks <- struct{}{}
					onDestory()
					close(events)
					close(acks)
					close(done)
					return
				case app.FrameEvent:
					gtx := app.NewContext(&ops, event)
					// fill the entire window with the background color
					paint.FillShape(gtx.Ops, th.Palette.Bg,
						clip.Rect{Max: gtx.Constraints.Min}.Op())
					// defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
					// paint.Fill(gtx.Ops, th.Palette.Bg)
					// render contents
					fn(w, gtx, th)
					// render frame
					event.Frame(gtx.Ops)
				}
				// If we didn't get a destroy event, ack that we're finished processing the window event
				// so that the other goroutine can continue.
				acks <- struct{}{}
			case <-refresh:
				// case newText := <-someChannel:
				// 	// ed.SetText(newTextefresh:
				// ed.SetText(newText)
				w.Invalidate()
			}
		}

	}()

	app.Main()
}

func Screenshot(gtx layout.Context, filename string) {
	sz := image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Max.Y}
	w, err := headless.NewWindow(sz.X, sz.Y)
	if err != nil {
		log.Println("[Screenshot] ERROR getting headless")
	}
	w.Frame(gtx.Ops)
	img := image.NewRGBA(image.Rectangle{Max: sz})
	if err := w.Screenshot(img); err != nil {
		log.Println("[Screenshot] ERROR getting screenshot")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Println("[Screenshot] ERROR encoding")
	}
	if err := os.WriteFile(filename, buf.Bytes(), 0o666); err != nil {
		log.Println("[Screenshot] ERROR saving file", filename)
	}
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/disintegration/imaging" // hmm, has lots of options...
	"github.com/nxshock/colorcrop"
)

type Config struct {
	Mode            string  `json:"mode"`
	ImageSize       int     `json:"image_size"` // for square
	WhiteThresold   float64 `json:"white_thresold"`
	MainSpacer      int     `json:"main_spacer"`
	ThumbnailSpacer int     `json:"thumbnail_spacer"`
	ThumbnailSize   int     `json:"thumbnail_size"` // for squares
	ThumbnailPos    int     `json:"thumbnail_pos"`
	DotScale        float64 `json:"dot_scale"`
	DotImage        string  `json:"dot_image"`
	ImageEnding     string  `json:"image_ending"`
	FlipImages      string  `json:"flip_images"`
}

func getConfig(cf string) (*Config, error) {
	f, err := os.Open(cf)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config Config
	if err = json.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func addMainImage(c *Config, filePath string, finalImg *image.RGBA) error {

	fmt.Printf("\rImporting image %d", 1)

	imgFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return err
	}
	if c.FlipImages != "no" {
		img = imaging.FlipH(img)
	}

	imgCropped := colorcrop.Crop(img, color.RGBA{215, 255, 255, 255}, c.WhiteThresold)
	dy := float64(imgCropped.Bounds().Dy())
	dx := float64(imgCropped.Bounds().Dx())

	fx, fy := 0.0, 0.0
	if c.Mode == "bottom" {
		fx = float64(c.ImageSize)
		fy = float64(c.ThumbnailPos - c.MainSpacer)
	} else {
		fx = float64(c.ThumbnailPos - c.MainSpacer)
		fy = float64(c.ImageSize)
	}

	// calculate factor to fit image into the available space
	f := 0.0
	align := ""
	if fx/dx < fy/dy {
		f = fx / dx
		align = "vertical"
	} else {
		f = fy / dy
		align = "horizontal"
	}
	imgSized := imaging.Resize(imgCropped, int(dx*f), int(dy*f), imaging.Lanczos)
	dx = float64(imgSized.Bounds().Dx())
	dy = float64(imgSized.Bounds().Dy())

	// calculate rect to align image (middle of available space plus half of the image)
	var r image.Rectangle
	if align == "vertical" {
		r = image.Rect(0, int(fy/2-dy/2), int(fx), int(fy/2+dy/2))
	} else {
		r = image.Rect(int(fx/2-dx/2), 0, int(fx/2+dx/2), int(fy))
	}
	draw.Draw(finalImg, r, imgSized, image.Point{0, 0}, draw.Src)

	return nil
}

func addThumbnails(c *Config, images []string, finalImg *image.RGBA) error {
	printDots := ""
	for i, path := range images {

		fmt.Printf("\rImporting image %d", i+1)

		if printDots != "" {
			path = printDots
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(file)
		if err != nil {
			return err
		}
		if c.FlipImages != "no" {
			img = imaging.FlipH(img)
		}
		imgCropped := colorcrop.Crop(img, color.RGBA{215, 255, 255, 255}, c.WhiteThresold)
		dy := float64(imgCropped.Bounds().Dy())
		dx := float64(imgCropped.Bounds().Dx())
		fx, fy := float64(c.ThumbnailSize), float64(c.ThumbnailSize)

		// calculate factor to fit image into the available space
		f := 0.0
		align := ""
		if fx/dx < fy/dy {
			f = fx / dx
			align = "vertical"
		} else {
			f = fy / dy
			align = "horizontal"
		}
		if printDots != "" {
			f *= c.DotScale
		}
		imgSized := imaging.Resize(imgCropped, int(dx*f), int(dy*f), imaging.Lanczos)
		dx = float64(imgSized.Bounds().Dx())
		dy = float64(imgSized.Bounds().Dy())

		var r image.Rectangle
		if c.Mode == "bottom" {
			if printDots != "" {
				// vertical and horizontal (dotScale...)
				r = image.Rect(
					(i+1)*c.ThumbnailSpacer+i*c.ThumbnailSize+int(fx/2-dx/2),
					c.ThumbnailPos+int(fy/2-dy/2),
					(i+1)*c.ThumbnailSpacer+(i+1)*c.ThumbnailSize+int(fx/2+dx/2)+int(fx),
					c.ThumbnailPos+c.ThumbnailSize+int(fy)+int(fy/2+dy/2),
				)
			} else if align == "vertical" {
				r = image.Rect(
					(i+1)*c.ThumbnailSpacer+i*c.ThumbnailSize,
					c.ThumbnailPos+int(fy/2-dy/2),
					(i+1)*c.ThumbnailSpacer+(i+1)*c.ThumbnailSize+int(fx),
					c.ThumbnailPos+c.ThumbnailSize+int(fy/2+dy/2),
				)
			} else {
				r = image.Rect(
					(i+1)*c.ThumbnailSpacer+i*c.ThumbnailSize+int(fx/2-dx/2),
					c.ThumbnailPos,
					(i+1)*c.ThumbnailSpacer+(i+1)*c.ThumbnailSize+int(fx/2+dx/2),
					c.ThumbnailPos+c.ThumbnailSize+int(fy),
				)
			}
		} else {
			if printDots != "" {
				r = image.Rect(
					c.ThumbnailPos+int(fx/2-dx/2),
					(i+1)*c.ThumbnailSpacer+i*c.ThumbnailSize+int(fy/2-dy/2),
					c.ThumbnailPos+c.ThumbnailSize+int(fx/2+dx/2)+int(fx),
					(i+1)*c.ThumbnailSpacer+(i+1)*c.ThumbnailSize+int(fy)+int(fy/2+dy/2),
				)
			} else if align == "vertical" {
				r = image.Rect(
					c.ThumbnailPos,
					(i+1)*c.ThumbnailSpacer+i*c.ThumbnailSize+int(fy/2-dy/2),
					c.ThumbnailPos+c.ThumbnailSize+int(fx),
					(i+1)*c.ThumbnailSpacer+(i+1)*c.ThumbnailSize+int(fy/2+dy/2),
				)
			} else {
				r = image.Rect(
					c.ThumbnailPos+int(fx/2-dx/2),
					(i+1)*c.ThumbnailSpacer+i*c.ThumbnailSize,
					c.ThumbnailPos+c.ThumbnailSize+int(fx/2+dx/2),
					(i+1)*c.ThumbnailSpacer+(i+1)*c.ThumbnailSize+int(fy),
				)

			}
		}
		draw.Draw(finalImg, r, imgSized, image.Point{0, 0}, draw.Src)

		// if not last and if next two do not fit, dots and stop
		if printDots != "" {
			break
		}
		if len(images[i+1:]) != 1 && (i+3)*c.ThumbnailSpacer+(i+3)*c.ThumbnailSize > c.ImageSize {
			printDots = c.DotImage
		}
	}
	return nil
}

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

func findImages(c *Config, path string) []string {
	// find dir
	var dir string
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalln(err)
	}
	for _, f := range files {
		if f.Name() == ".git" {
			continue
		}
		if isDir, _ := isDirectory("./" + f.Name()); isDir {
			if dir != "" {
				log.Fatalln("Fehler. Es darf hier nur einen Ordner mit Bildern geben.")
			}
			dir = f.Name() + "/"
		}
	}

	// find images
	var images []string
	files, err = ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalln(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), c.ImageEnding) || strings.HasSuffix(f.Name(), strings.ToUpper(c.ImageEnding)) {
			images = append(images, dir+f.Name())
		}
	}
	if len(images) <= 0 {
		log.Fatalf("Fehler. Es wurden keine Bilder mit der Endung '%s', in '%s' gefunden.", c.ImageEnding, dir)
	}
	sort.Strings(images)

	return images
}

func main() {

	c := flag.String("config", "", "flag config (ebay or shopify) is required")
	flag.Parse()

	cf := ""
	if *c == "ebay" {
		cf = "config_ebay.json"
	}
	if *c == "shopify" {
		cf = "config_shopify.json"
	}
	if cf == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config, err := getConfig(cf)
	if err != nil {
		log.Fatalf("Config konnte nicht gelesen werden: %v\n", err)
	}

	finalImg := image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{config.ImageSize, config.ImageSize},
		},
	)
	draw.Draw(finalImg, finalImg.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.ZP, draw.Src)

	images := findImages(config, "./")
	if err := addMainImage(config, images[0], finalImg); err != nil {
		log.Fatalln(err)
	}
	if err := addThumbnails(config, images[1:], finalImg); err != nil {
		log.Fatalln(err)
	}

	out, err := os.Create("./output.jpg")
	if err != nil {
		log.Fatalln(err)
	}
	jpeg.Encode(out, finalImg, &jpeg.Options{Quality: 95})

	fmt.Println("\r                                  ")
	fmt.Println("Erfolg. Startbild wurde geschrieben.")
	fmt.Println("")
}

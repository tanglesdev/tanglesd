package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"

	"github.com/mitchellh/cli"
	yall "yall.in"

	"tangl.es/code/blobs"
	"tangl.es/code/images"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

func imagesProcessCommandFactory(ui cli.Ui, processor images.Processor, listener images.Listener, b blobs.Storer, log *yall.Logger) func() (cli.Command, error) {
	return func() (cli.Command, error) {
		return imagesProcessCommand{
			ui:        ui,
			processor: processor,
			listener:  listener,
			blobs:     b,
			log:       log,
		}, nil
	}
}

type imagesProcessCommand struct {
	ui        cli.Ui
	processor images.Processor
	listener  images.Listener
	blobs     blobs.Storer
	log       *yall.Logger
}

func (u imagesProcessCommand) Help() string {
	return `Run image processing.

Start up the long-running daemon that will perform on-the-fly processing
on new images that are uploaded.`
}

func (u imagesProcessCommand) Run(args []string) int {
	ctx := context.Background()
	ctx = yall.InContext(ctx, u.log)
	err := u.listener.Listen(ctx, u.processImage)
	if err != nil {
		u.ui.Error(fmt.Sprintf("Error listening: %s", err.Error()))
		return 1
	}
	return 0
}

func (u imagesProcessCommand) Synopsis() string {
	return "Set up your tangl.es credentials."
}

func (u imagesProcessCommand) processImage(ctx context.Context, source string) (images.Image, error) {
	rc, err := u.blobs.Download(ctx, source)
	if err != nil {
		return images.Image{}, err
	}
	defer rc.Close()
	img, b, err := u.processor.Process(ctx, rc)
	if err != nil {
		return images.Image{}, err
	}
	sha := sha256.New()
	_, err = sha.Write(b)
	if err != nil {
		return images.Image{}, err
	}
	img.SHA256 = hex.EncodeToString(sha.Sum(nil))
	img.SourceSHA256 = source

	crc := crc32.New(crc32cTable)
	_, err = crc.Write(b)
	if err != nil {
		return images.Image{}, err
	}
	wc, err := u.blobs.Upload(ctx, img.SHA256+"."+img.Extension, binary.BigEndian.Uint32(crc.Sum(nil)))
	if err != nil {
		return images.Image{}, err
	}
	if wc == nil {
		// this file already exists
		return img, nil
	}
	defer wc.Close()
	_, err = wc.Write(b)
	if err != nil {
		return images.Image{}, err
	}
	return img, nil
}

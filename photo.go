package main

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log"
	"time"
)

const EVENT_IMG_HOST = "https://bmkg-content-inatews.storage.googleapis.com/"

func sendPhoto(ctx context.Context, b *bot.Bot, path string, caption string) {
	if len(path) < 1 {
		return
	}

	eventID_When_This := currentEventID
	for {
		if currentEventID != eventID_When_This {
			log.Printf("sendPhoto(%s): Current Event ID has changed. Breaking loop....", path)
			break
		}

		photoURL := EVENT_IMG_HOST + path

		resp, err := http_get(photoURL)
		if err != nil {
			log.Printf("sendPhoto(%s): Failed to fetch photo. Retrying in 15s....: %s", path, err)
			time.Sleep(time.Second * 15)
			continue
		}

		defer resp.Body.Close()

		// Sometime sending image source URL on sendPhoto just don't works anymore.
		// It will give you this weird error: Bad Request: wrong file identifier/HTTP URL specified
		// Even the URL and file identifier is valid.
		photo := models.InputFileUpload{
			Filename: path,
			Data:     resp.Body,
		}

		if _, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:    config.ChatID,
			Photo:     &photo,
			Caption:   caption,
			ParseMode: models.ParseModeMarkdownV1,
		}); err != nil {
			log.Printf("sendPhoto(%s): bot: Failed to send photo. Retrying in 15s....: %s", path, err)
			time.Sleep(time.Second * 15)
			continue
		}

		break
	}
}

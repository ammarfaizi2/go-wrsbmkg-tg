package main

import (
	"codeberg.org/Yonle/go-wrsbmkg"
	"codeberg.org/Yonle/go-wrsbmkg/helper"
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"net/http"
	"strconv"
	"time"
)

func startBMKG(ctx context.Context, b *bot.Bot) {
	p := wrsbmkg.Penerima{
		Gempa:    make(chan wrsbmkg.DataJSON),
		Realtime: make(chan wrsbmkg.DataJSON),
		Narasi:   make(chan string),

		Interval: time.Second * 1,
		API_URL:  wrsbmkg.DEFAULT_API_URL,

		HTTP_Client: http.Client{
			Timeout: time.Second * 30,
		},
	}

	p.MulaiPolling(ctx)

listener:
	for {
		select {
		case g := <-p.Gempa:
			gempa := helper.ParseGempa(g)
			msg := fmt.Sprintf(
				"*%s*\n\n%s\n\n%s\n\n%s\n\n%s\n",
				gempa.Subject,
				gempa.Description,
				gempa.Area,
				gempa.Potential,
				gempa.Instruction,
			)

			shakemapURL := "https://bmkg-content-inatews.storage.googleapis.com/" + gempa.Shakemap

			_, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:    config.ChatID,
				Caption:   msg,
				ParseMode: models.ParseModeMarkdownV1,
				Photo:     &models.InputFileString{Data: shakemapURL},
			})
			if err != nil {
				fmt.Println(err)
			}
		case r := <-p.Realtime:
			realtime := helper.ParseRealtime(r)
			t, _ := time.Parse(time.DateTime, realtime.Time)
			ft := t.Format(time.Kitchen)
			msg := fmt.Sprintf(
				"*%s*\n"+
					"`"+
					"Waktu     : %s\n"+
					"Magnitudo : M%.1f\n"+
					"Kedalaman : %.1f KM\n"+
					"Fase      : %v\n"+
					"Status    : %s"+
					"`",
				realtime.Place,
				ft,
				realtime.Magnitude,
				realtime.Depth,
				realtime.Phase,
				realtime.Status,
			)

			lat, _ := strconv.ParseFloat(realtime.Coordinates[1].(string), 64)
			long, _ := strconv.ParseFloat(realtime.Coordinates[0].(string), 64)

			venueTitle := fmt.Sprintf("M%.1f, %s", realtime.Magnitude, ft)

			m, err := b.SendVenue(ctx, &bot.SendVenueParams{
				ChatID:              config.ChatID,
				DisableNotification: true,
				Latitude:            lat,
				Longitude:           long,
				Title:               realtime.Place,
				Address:             venueTitle,
			})

			if err != nil {
				fmt.Println(err)
				continue listener
			}

			if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:              config.ChatID,
				Text:                msg,
				ParseMode:           models.ParseModeMarkdownV1,
				DisableNotification: true,
				ReplyParameters:     &models.ReplyParameters{MessageID: m.ID},
			}); err != nil {
				fmt.Println(err)
				continue listener
			}
		case n := <-p.Narasi:
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    config.ChatID,
				Text:      n,
				ParseMode: models.ParseModeHTML,
			})

			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

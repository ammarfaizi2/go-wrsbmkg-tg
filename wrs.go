package main

import (
	"codeberg.org/Yonle/go-wrsbmkg"
	"codeberg.org/Yonle/go-wrsbmkg/helper"
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log"
	"strconv"
	"time"
)

var WIB = time.FixedZone("WIB", +7*60*60)
var currentEventID string

func startBMKG(ctx context.Context, b *bot.Bot) {
	p := wrsbmkg.BuatPenerima()

	p.MulaiPolling(ctx)

listener:
	for {
		select {
		case g := <-p.Gempa:
			gempa := helper.ParseGempa(g)
			currentEventID = gempa.EventID

			msg := fmt.Sprintf(
				"*%s*\n\n%s\n\n%s\n\n%s\n\n%s\n",
				gempa.Subject,
				gempa.Description,
				gempa.Area,
				gempa.Potential,
				gempa.Instruction,
			)

			log.Printf("wrs: Got event ID: %s", gempa.EventID)
			log.Printf(gempa.Headline)

			// send headline first. As the shakemap isn't really ready at the time of the incident.
			if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: config.ChatID,
				Text:   gempa.Headline,
			}); err != nil {
				log.Printf("bot: Failed to send headline: %s", err)
			}

			go sendPhoto(ctx, b, gempa.Shakemap, msg)

			// Below code is for handling Tsunami warning.
			go sendPhoto(ctx, b, gempa.WZMap, "")
			go sendPhoto(ctx, b, gempa.TTMap, "")
			go sendPhoto(ctx, b, gempa.SSHMap, "")

			var zonaPeringatanText string

			for _, area := range gempa.WZAreas {
				zonaPeringatanText += fmt.Sprintf(
					"- %s: %s, %s\n",
					area.Level,
					area.Province,
					area.District,
				)
			}

			if len(zonaPeringatanText) > 0 {
				zonaPeringatanText += fmt.Sprintf(
					"\n*Instruksi*\n1. %s\n2. %s\n3. %s",
					gempa.Instruction1,
					gempa.Instruction2,
					gempa.Instruction3,
				)

				zonaPeringatanText = "*Zona-Zona Peringatan*\n" + zonaPeringatanText

				fmt.Println(zonaPeringatanText)

				if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    config.ChatID,
					Text:      zonaPeringatanText,
					ParseMode: models.ParseModeMarkdownV1,
				}); err != nil {
					log.Printf("bot: Failed to send zonaPeringatanText: %s", err)
				}
			}
		case r := <-p.Realtime:
			realtime := helper.ParseRealtime(r)
			t, _ := time.Parse(time.DateTime, realtime.Time)
			tl := t.In(WIB)
			date := tl.Format(time.DateOnly)
			ft := tl.Format(time.Kitchen)
			msg := fmt.Sprintf(
				"*M%.1f* %s\n"+
					"`"+
					"Tanggal   : %s\n"+
					"Waktu     : %s\n"+
					"Kedalaman : %.1f KM\n"+
					"Fase      : %v\n"+
					"Status    : %s"+
					"`",
				realtime.Magnitude,
				realtime.Place,
				date,
				ft,
				realtime.Depth,
				realtime.Phase,
				realtime.Status,
			)

			log.Printf("wrs: Got realtime info: M%.1f %s", realtime.Magnitude, realtime.Place)

			lat, _ := strconv.ParseFloat(realtime.Coordinates[1].(string), 64)
			long, _ := strconv.ParseFloat(realtime.Coordinates[0].(string), 64)

			venueTitle := fmt.Sprintf("M%.1f, %s %s", realtime.Magnitude, date, ft)

			m, err := b.SendVenue(ctx, &bot.SendVenueParams{
				ChatID:              config.ChatID,
				DisableNotification: true,
				Latitude:            lat,
				Longitude:           long,
				Title:               realtime.Place,
				Address:             venueTitle,
			})

			if err != nil {
				log.Printf("bot: Failed to send venue: %s", err)
				continue listener
			}

			if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:              config.ChatID,
				Text:                msg,
				ParseMode:           models.ParseModeMarkdownV1,
				DisableNotification: true,
				ReplyParameters:     &models.ReplyParameters{MessageID: m.ID},
			}); err != nil {
				log.Printf("bot: Failed to send realtime info message: %s", err)
				continue listener
			}
		case n := <-p.Narasi:
			narasi := helper.CleanNarasi(n)
			log.Println("wrs: Got narasi")

			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: config.ChatID,
				Text:   narasi,
			})

			if err != nil {
				log.Printf("bot: Failed to send narasi: %s", err)
			}
		}
	}
}

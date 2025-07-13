package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeberg.org/Yonle/go-wrsbmkg"
	"codeberg.org/Yonle/go-wrsbmkg/helper"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var WIB = time.FixedZone("WIB", +7*60*60)
var currentEventID string

/**
 * A simple file-based memory to keep track of sent messages with
 * the help of filesystem. Hash the message, check if the file exists
 * in the memory directory, if it exists, skip the message. If it does
 * not exist, create the file to remember the message.
 *
 * @param mutex		A mutex to protect the memory directory.
 * @param data		The message data to be checked and remembered.
 * @return		true if the message should be sent
 *			false if it should be skipped.
 */
func IsNewMessage(m *sync.Mutex, data string) bool {
	/*
	 * Always send the message if the memory directory is not set.
	 */
	if len(config.MsgMemoryDir) == 0 {
		return true
	}

	md5sum := md5.Sum([]byte(data))
	fpath := fmt.Sprintf("%s/%x", config.MsgMemoryDir, md5sum)

	m.Lock()
	defer m.Unlock()
	/*
	 * If the file exists, skip the message. We've already sent it.
	 */
	if _, err := os.Stat(fpath); err == nil {
		log.Printf("wrs: Skipping message, already sent: %x", md5sum)
		return false
	}

	if err := os.MkdirAll(config.MsgMemoryDir, 0755); err != nil && !os.IsExist(err) {
		log.Printf("Failed to create message memory directory: %s", err)
		return true
	}

	/*
	 * Create the file to remember the message.
	 */
	f, err := os.Create(fpath)
	if err != nil {
		log.Printf("Failed to create message memory file: %s", err)
		return true
	}
	f.Close()

	return true
}

func ScanAndDeleteOldMessages(m *sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	files, err := os.ReadDir(config.MsgMemoryDir)
	if err != nil {
		log.Printf("Failed to read message memory directory: %s", err)
		return
	}

	for _, file := range files {
		st, err := file.Info()
		if err != nil {
			log.Printf("Failed to get file info: %s", err)
			continue
		}

		if time.Since(st.ModTime()) <= 7*24*time.Hour {
			continue
		}

		fpath := fmt.Sprintf("%s/%s", config.MsgMemoryDir, file.Name())
		if err := os.Remove(fpath); err != nil {
			log.Printf("Failed to delete old message memory file: %s", err)
		} else {
			log.Printf("Deleted old message memory file: %s", fpath)
		}
	}
}

func MemDirHouseKeeping(m *sync.Mutex) {
	for {
		log.Println("wrs: Starting message memdir housekeeping")
		ScanAndDeleteOldMessages(m)
		log.Println("wrs: Finished message memdir housekeeping, will be back in 3 hours")
		time.Sleep(3 * time.Hour)
	}
}

func startBMKG(ctx context.Context, b *bot.Bot) {
	p := wrsbmkg.BuatPenerima()

	p.MulaiPolling(ctx)

	mu := sync.Mutex{}
	if len(config.MsgMemoryDir) > 0 {
		go MemDirHouseKeeping(&mu)
	}

listener:
	for {
		select {
		case g := <-p.Gempa:
			gempa := helper.ParseGempa(g)
			currentEventID = gempa.EventID

			if config.MinMag >= gempa.Magnitude {
				continue listener
			}

			if len(config.FilterRegion) > 0 && !strings.Contains(
				strings.ToLower(gempa.Area), config.FilterRegion,
			) {
				continue listener
			}

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

			if !IsNewMessage(&mu, msg) {
				continue listener
			}

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
					"- %s: %s, %s (estimasi waktu tiba: %s %s)\n",
					area.Level,
					area.Province,
					area.District,
					area.Date,
					area.Time,
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

			if config.MinMag >= realtime.Magnitude {
				continue listener
			}

			if len(config.FilterRegion) > 0 && !strings.Contains(
				strings.ToLower(realtime.Place), config.FilterRegion,
			) {
				continue listener
			}

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

			if !IsNewMessage(&mu, msg) {
				continue listener
			}

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

			if !IsNewMessage(&mu, narasi) {
				continue listener
			}

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

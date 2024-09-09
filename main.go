package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
)

func main() {
	ReadConfig("wrsbmkg_telegrambot_config.yaml")

	if config.ChatID == 0 {
		panic("ChatID is not provided!")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{}

	b, err := bot.New(config.Token, opts...)
	if err != nil {
		panic(err)
	}

	log.Println("Ready. New reports will be ready in 15 seconds...")

	go startBMKG(ctx, b)
	b.Start(ctx)
}

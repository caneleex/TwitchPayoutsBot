package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/log"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	apiUrl    = "https://twitchpayouts.com/api/payouts"
	payoutMap = make(map[string]PayoutEntry)
)

func main() {
	var payouts PayoutJson

	response, err := http.Get(apiUrl)
	if err != nil {
		panic(err)
	}
	closer := response.Body
	body, err := io.ReadAll(closer)
	err = closer.Close()
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(body, &payouts)
	if err != nil {
		panic(err)
	}
	for _, entry := range payouts.Payouts {
		username := entry.Username
		if username != "__unknown__" {
			payoutMap[strings.ToLower(username)] = entry
		}
	}
	log.Infof("loaded %d payouts.", len(payoutMap))

	log.SetLevel(log.LevelInfo)
	log.Info("starting the bot...")
	log.Info("disgo version: ", disgo.Version)

	client, err := disgo.New(os.Getenv("PAYOUTS_TOKEN"),
		bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentsNone)),
		bot.WithCacheConfigOpts(cache.WithCacheFlags(cache.FlagsNone)),
		bot.WithEventListeners(&events.ListenerAdapter{
			OnApplicationCommandInteraction: onCommand,
		}))
	if err != nil {
		log.Fatal("error while building disgo instance: ", err)
	}

	defer client.Close(context.TODO())

	if client.OpenGateway(context.TODO()) != nil {
		log.Fatalf("error while connecting to the gateway: %s", err)
	}

	log.Infof("payouts bot is now running.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-s
}

func onCommand(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	name := data.CommandName()
	if name == "payout" {
		user := data.String("creator")
		payout, ok := payoutMap[strings.ToLower(user)]
		messageBuilder := discord.NewMessageCreateBuilder()
		if !ok {
			event.CreateMessage(messageBuilder.
				SetContentf("No payout found for user **%s**.", user).
				SetEphemeral(true).
				Build())
			return
		}
		embedBuilder := discord.NewEmbedBuilder()
		username := payout.Username
		embedBuilder.SetAuthor(username, "https://twitch.tv/"+username, payout.AvatarURL)

		p := message.NewPrinter(language.AmericanEnglish)
		earning := payout.GrossEarning
		formatted := p.Sprintf("%0.f", earning)
		embedBuilder.SetDescriptionf("Gross Earnings: **$%s** (**$%.2f**)", formatted, earning)
		embedBuilder.SetFooterText(fmt.Sprintf("User ID: %s | twitchpayouts.com", *getUserId(payout)))
		event.CreateMessage(messageBuilder.
			SetEmbeds(embedBuilder.Build()).
			SetEphemeral(true).
			Build())
	}
}

func getUserId(entry PayoutEntry) *string {
	userId := entry.UserId
	switch userId := userId.(type) {
	case string:
		return &userId
	case float64:
		str := fmt.Sprintf("%.0f", userId)
		return &str
	}
	return nil
}

type PayoutJson struct {
	Payouts []PayoutEntry `json:"default"`
}

type PayoutEntry struct {
	Rank         int         `json:"rank"`
	Username     string      `json:"username"`
	UserId       interface{} `json:"user_id"`
	GrossEarning float64     `json:"gross_earning"`
	AvatarURL    string      `json:"pfp"`
}

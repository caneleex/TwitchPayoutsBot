package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
}

func getUserId(entry PayoutEntry) *string {
	userId := entry.UserId
	switch userId := userId.(type) {
	case string:
		return &userId
	case float64:
		str := fmt.Sprintf("%f", userId)
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

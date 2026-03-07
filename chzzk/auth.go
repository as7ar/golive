package chzzk

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type liveStatusResponse struct {
	Content struct {
		ChatChannelId string `json:"chatChannelId"`
	} `json:"content"`
}

type accessTokenResponse struct {
	Content struct {
		AccessToken string `json:"accessToken"`
		ExtraToken  string `json:"extraToken"`
	} `json:"content"`
}

func GetChatChannelId(id string) (string, error) {
	url := "https://api.chzzk.naver.com/polling/v2/channels/" + id + "/live-status"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("API_CHAT_CHANNEL_ID_ERROR")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result liveStatusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Content.ChatChannelId, nil
}

func GetAccessToken(chatChannelId string) (string, error) {
	url := "https://comm-api.game.naver.com/nng_main/v1/chats/access-token?channelId=" + chatChannelId + "&chatType=STREAMING"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("API_ACCESS_TOKEN_ERROR")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result accessTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Content.AccessToken + ";" + result.Content.ExtraToken, nil
}

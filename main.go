package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".envファイルが読み込めません。サンプルは.env.sampleです。")
	}

	misskeyHost := os.Getenv("MISSKEY_HOST")
	accessToken := os.Getenv("MISSKEY_ACCESS_TOKEN")

	//ここで何故か空のパーミッションになってしまい、認証が通らない。
	if accessToken == "" {
		sessionID := generateUUID()
		authURL := fmt.Sprintf("https://%s/miauth/%s?name=MyApp&callback=http%%3A%%2F"+
			"%%2Flocalhost%%3A3000%%2Fcallback&permission=write:admin:suspend-user",
			misskeyHost, sessionID) //コールバック後に任意のメッセージを表示するように実装したい

		fmt.Printf("ブラウザで認証してください。:\n%s\n", authURL)

		fmt.Println("処理待ち中。認証が終わったらエンターを押してください。")
		_, _ = fmt.Scanln()

		accessToken, _, err = getAccessToken(misskeyHost, sessionID)
		if err != nil {
			log.Fatal("アクセストークンの取得に失敗しました:", err)
		}

		err = os.Setenv("MISSKEY_ACCESS_TOKEN", accessToken)
		if err != nil {
			log.Fatal("アクセストークンを.envに書き込むのに失敗しました:", err)
		}

		fmt.Println("アクセストークンを.envに書き込みました。次回からはこのアクセストークンを使用します。")
	} else {
		fmt.Println("アクセストークンを.envから読み込みました。")
	}

	apiEndpoint := fmt.Sprintf("https://%s/api/admin/suspend-user", misskeyHost)

	suspendThreshold := time.Now().Add(-1 * time.Minute) // 直近1分間に投稿したユーザーをサスペンド

	suspendUsers(apiEndpoint, accessToken, suspendThreshold)
}

func generateUUID() string {
	uuid := uuid.New()
	return uuid.String()
}

func getAccessToken(misskeyHost, sessionID string) (string, map[string]interface{}, error) {
	tokenURL := fmt.Sprintf("https://%s/api/miauth/%s/check", misskeyHost, sessionID)

	resp, err := http.Post(tokenURL, "application/json", nil)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", nil, err
	}

	return result["token"].(string), result["user"].(map[string]interface{}), nil
}

func suspendUsers(apiEndpoint, accessToken string, threshold time.Time) {
	payload := map[string]interface{}{
		"threshold": threshold.Format(time.RFC3339),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Println(err)
		return
	}

	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		if err.Error() == "サーバーが HTTP レスポンスをHTTPS に投げてもうた。" {
			log.Println("HTTPからHTTPSへのリダイレクトのエラーを無視します。")
			return
		}

		log.Println(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("サスペンドが成功しました。")
	} else {
		log.Printf("サスペンドに失敗しました: %d\n", resp.StatusCode)
	}
}

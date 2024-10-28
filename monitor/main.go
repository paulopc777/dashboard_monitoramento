package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Auth struct {
	Token string `json:"token"`
}

func Authentication(auth *Auth) error {

	loginData := map[string]string{
		"email":    os.Getenv("USER"),
		"password": os.Getenv("PASS"),
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("erro ao codificar dados de login: %v", err)
	}
	res, err := http.Post("https://api-v2.monitchat.com/api/v1/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao fazer login no MonitChat: %v", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler resposta do login: %v", err)
	}
	var result map[string]interface{}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("erro ao decodificar resposta JSON: %v", err)
	}

	if token, ok := result["access_token"].(string); ok {
		auth.Token = token
	} else {
		return fmt.Errorf("access_token não encontrado na resposta")
	}
	return nil
}

type SocialAccount struct {
	Active      int    `json:"connected"`
	PhoneNumber string `json:"phone_number"`
}

type MonitorData struct {
	Name           string          `json:"name"`
	SocialAccounts []SocialAccount `json:"social_accounts"`
}

type ResultData struct {
	Name        string          `json:"name"`
	PhoneNumber []SocialAccount `json:"phone_number"`
}

func GetMonitorData(token string) ([]ResultData, error) {
	req, err := http.NewRequest("GET", "https://api-v2.monitchat.com/api/v1/social/whatsapp/monitor", nil)
	if err != nil {
		fmt.Printf("Erro ao criar requisição")
	}

	tokenBear := "Bearer" + token
	req.Header.Set("authorization", tokenBear)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		fmt.Print("erro ao fazer requisição do monitor")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Erro ao ler Body da requisição")
	}

	var response struct {
		Data []MonitorData `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar o JSON: %v", err)
	}
	var results []ResultData
	for _, item := range response.Data {
		results = append(results, ResultData{
			Name:        item.Name,
			PhoneNumber: item.SocialAccounts,
		})
	}

	return results, nil
}

func HandleMetrics(token string) func(*gin.Context) {
	return func(c *gin.Context) {

		data, err := GetMonitorData(token)
		if err != nil {
			c.String(http.StatusInternalServerError, "Erro ao obter dados de monitoramento: %v", err)
			return
		}

		// Formata as métricas para Prometheus
		var metrics string
		for _, item := range data {
			for _, account := range item.PhoneNumber {
				metrics += fmt.Sprintf(
					"monitchat_whatsapp_active{name=\"%s\", phone_number=\"%s\"} %d\n",
					item.Name, account.PhoneNumber, account.Active,
				)
			}
		}

		// Define o conteúdo como "text/plain" para o Prometheus
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(metrics))
	}
}

func main() {
	auth := &Auth{}
	if err := godotenv.Load(".env"); err != nil {
		panic("err open .env")
	}

	if err := Authentication(auth); err != nil {
		fmt.Printf("error %v", err)
	} else {
		fmt.Printf("Token de autenticação: %s\n", auth.Token)
	}

	r := gin.Default()
	r.GET("/api/metrics", HandleMetrics(auth.Token))
	r.Run(":4080")
}

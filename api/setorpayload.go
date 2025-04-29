package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// TangerinoWorkplacePayload representa a estrutura do JSON esperado pela API do Tangerino para setores.
type TangerinoWorkplacePayload struct {
	ExternalId string `json:"externalId"`
	Name       string `json:"name"`
}

// PostWorkplaceToTangerino envia os dados do setor para a API do Tangerino.
func PostWorkplaceToTangerino(payload TangerinoWorkplacePayload) error {
	apiURL := "https://employer.tangerino.com.br/api/employer/workplace/register?allowUpdate=false"
	apiKey := os.Getenv("EmployerApiToken")

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload do setor: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição POST para setor: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar requisição para API do Tangerino (setor): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Setor CODSETOR=%s enviado com sucesso para o Tangerino.\n", payload.ExternalId)
		return nil
	} else {
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			log.Printf("Erro ao decodificar resposta da API (setor): %v\n", err)
		}
		return fmt.Errorf("erro ao enviar setor para API do Tangerino. Status Code: %d, Response: %v", resp.StatusCode, result)
	}
}

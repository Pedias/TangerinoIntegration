package company

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// CompanyPayload representa o payload para envio de dados da filial
type CompanyPayload struct {
	Cnpj            string `json:"cnpj"`               // CNPJ da filial
	CnpjMask        string `json:"cnpjMask,omitempty"` // Não utilizado (omitido)
	DescriptionName string `json:"descriptionName"`    // Pode ser uma descrição (opcional)
	ExternalId      string `json:"externalId"`         // Usamos CODFILIAL
	FantasyName     string `json:"fantasyName"`        // Nome Fantasia
	//Id              int    `json:"id"`               // CODFILIAL convertido para inteiro
	SocialReason string `json:"socialReason"` // Razão Social
}

// CompanyEndpoint é o endpoint para enviar dados da filial
const CompanyEndpoint = "https://employer.tangerino.com.br/api/employer/companies"

// getEmployerApiToken recupera o token da API a partir das variáveis de ambiente
func getEmployerApiToken() string {
	token := os.Getenv("EmployerApiToken")
	return token
}

// PostCompanyToTangerino envia o payload de uma filial para o endpoint da API
func PostCompanyToTangerino(cp CompanyPayload) error {
	bodyBytes, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("erro ao converter payload para JSON: %w", err)
	}

	req, err := http.NewRequest("POST", CompanyEndpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("erro ao criar request POST: %w", err)
	}

	req.Header.Set("Authorization", getEmployerApiToken())
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar request POST: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("falha no envio da filial. HTTP %d", resp.StatusCode)
	}

	return nil
}

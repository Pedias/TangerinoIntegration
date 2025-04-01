package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func getEmployerApiToken() string {
	token := os.Getenv("EmployerApiToken")
	return token
}

// RegisterEndpoint é o endpoint base para inserção
const RegisterEndpoint = "https://employer.tangerino.com.br/api/employer/employee/register"

// TangerinoEmployeePayload representa os dados que serão enviados à Employer API
type TangerinoEmployeePayload struct {
	Name          string `json:"name"`
	Cpf           string `json:"cpf"`
	AdmissionDate string `json:"admissionDate"`
	EffectiveDate string `json:"effectiveDate"`
	Email         string `json:"email,omitempty"`
	ExternalId    string `json:"externalId"`
	BirthDate     string `json:"birthDate,omitempty"`
	Carteiratrab  string `json:"ctps"`
	Seriecarttrab string `json:"series"`
	Pispasep      string `json:"pis"`
	Telefone      string `json:"cellphone,omitempty"`
	Cargo         string `json:"jobRoleDescription"`
	Gender        string `json:"gender,omitempty"`
	Intern        bool   `json:"intern"`
	Company       int    `json:"company"`
}

// PostEmployeeToTangerino envia o payload para inserção (endpoint padrão)
func PostEmployeeToTangerino(emp TangerinoEmployeePayload) error {
	// Converte struct para JSON
	bodyBytes, err := json.Marshal(emp)
	if err != nil {
		return fmt.Errorf("erro ao converter struct para JSON: %w", err)
	}

	// Monta a requisição
	req, err := http.NewRequest("POST", RegisterEndpoint, bytes.NewBuffer(bodyBytes))
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("falha na inserção. HTTP %d", resp.StatusCode)
	}

	return nil
}

// PostEmployeeToTangerinoUpdate envia o payload para atualização (usando allowUpdate=true)
func PostEmployeeToTangerinoUpdate(emp TangerinoEmployeePayload) error {
	// Adiciona o parâmetro allowUpdate=true na URL
	updateURL := RegisterEndpoint + "?allowUpdate=true"

	bodyBytes, err := json.Marshal(emp)
	if err != nil {
		return fmt.Errorf("erro ao converter struct para JSON: %w", err)
	}

	req, err := http.NewRequest("POST", updateURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("erro ao criar request POST para update: %w", err)
	}

	req.Header.Set("Authorization", getEmployerApiToken())
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar request POST para update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("falha na atualização. HTTP %d", resp.StatusCode)
	}

	return nil
}

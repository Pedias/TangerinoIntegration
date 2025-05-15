package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// DismissEmployeePayload corresponde ao EmployeeDismissDTO do Swagger
type DismissEmployeePayload struct {
	ExternalId      string `json:"externalId"`
	ResignationDate int64  `json:"resignationDate"` // obrigatório
}

const dismissURL = "https://employer.tangerino.com.br/api/employer/employee/dismiss"

// DismissEmployee envia o pedido de demissão para a API
func DismissEmployee(p DismissEmployeePayload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload de demissão: %w", err)
	}
	req, err := http.NewRequest("POST", dismissURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("erro ao criar request de demissão: %w", err)
	}
	req.Header.Set("Authorization", getEmployerApiToken())
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar request de demissão: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("falha na demissão. HTTP %d", resp.StatusCode)
	}
	return nil
}

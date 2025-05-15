// api/tangerino.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func getEmployerApiToken() string {
	return os.Getenv("EmployerApiToken")
}

const RegisterEndpoint = "https://employer.tangerino.com.br/api/employer/employee/register"

type TangerinoEmployeePayload struct {
	Name                string `json:"name"`
	Cpf                 string `json:"cpf"`
	AdmissionDate       string `json:"admissionDate"`
	EffectiveDate       string `json:"effectiveDate"`
	ExternalId          string `json:"externalId"`
	Email               string `json:"email,omitempty"`
	BirthDate           string `json:"birthDate"`
	Carteiratrab        string `json:"ctps"`
	Seriecarttrab       string `json:"series"`
	Pispasep            string `json:"pis"`
	Telefone            string `json:"cellphone"`
	Cargo               string `json:"jobRoleDescription"`
	Gender              string `json:"gender"`
	Intern              bool   `json:"intern"`
	Company             string `json:"company"`
	Chapa               string `json:"companyExternalId"`
	WorkplaceExternalId string `json:"workplaceExternalId"`
}

func postEmployee(url string, emp TangerinoEmployeePayload) error {
	body, _ := json.Marshal(emp)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", getEmployerApiToken())
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func PostEmployeeToTangerino(emp TangerinoEmployeePayload) error {
	return postEmployee(RegisterEndpoint, emp)
}

func PostEmployeeToTangerinoUpdate(emp TangerinoEmployeePayload) error {
	return postEmployee(RegisterEndpoint+"?allowUpdate=true", emp)
}

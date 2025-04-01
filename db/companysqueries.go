package db

import (
	"TangerinoIntegration/models"
	"database/sql"
	"fmt"
	"log"
)

func GetTangerinoCompanies(conn *sql.DB) ([]models.TangerinoCompany, error) {
	query := `SELECT CODFILIAL, RAZAOSOCIAL, NOMEFANTASIA, CNPJ FROM RM.TANGERINO_COMPANY`
	rows, err := conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar query para companies: %w", err)
	}
	defer rows.Close()

	var companies []models.TangerinoCompany
	for rows.Next() {
		var c models.TangerinoCompany
		if err := rows.Scan(&c.CodFilial, &c.RazaoSocial, &c.NomeFantasia, &c.Cnpj); err != nil {
			return nil, fmt.Errorf("erro ao fazer scan dos dados da company: %w", err)
		}
		companies = append(companies, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração das linhas: %w", err)
	}
	log.Printf("Foram encontrados %d registros na view TANGERINO_COMPANY.\n", len(companies))
	return companies, nil
}

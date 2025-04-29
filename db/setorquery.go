package db

import (
	"TangerinoIntegration/models"
	"database/sql"
	"fmt"
	"log"
)

// GetTangerinoWorkplaces busca os dados dos setores na view TANGERINO_SETOR.
func GetTangerinoWorkplaces(conn *sql.DB) ([]models.TangerinoWorkplace, error) {
	query := `SELECT CODSETOR, NOMESETOR FROM RM.TANGERINO_SETOR`
	rows, err := conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar query para setores: %w", err)
	}
	defer rows.Close()

	var workplaces []models.TangerinoWorkplace
	for rows.Next() {
		var w models.TangerinoWorkplace
		if err := rows.Scan(&w.CodSetor, &w.NomeSetor); err != nil {
			return nil, fmt.Errorf("erro ao fazer scan dos dados do setor: %w", err)
		}
		workplaces = append(workplaces, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração das linhas: %w", err)
	}
	log.Printf("Foram encontrados %d registros na view TANGERINO_SETOR.\n", len(workplaces))
	return workplaces, nil
}

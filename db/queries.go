package db

import (
	"TangerinoIntegration/models"
	"database/sql"
	"fmt"
	"log"
)

func GetTangerinoUsers(conn *sql.DB) ([]models.TangerinoUser, error) {
	query := `SELECT CHAPA, NOME, SEXO, CPF, FUNCAO, NASCIMENTO, EMAIL, ADMISSAO, CARTEIRATRAB,
	 SERIECARTTRAB, PISPASEP, TELEFONE1, IDCOMPANY
		FROM RM.TANGERINO_USERS`

	rows, err := conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar SELECT na view: %w", err)
	}
	defer rows.Close()

	var usuarios []models.TangerinoUser
	for rows.Next() {
		var u models.TangerinoUser
		err := rows.Scan(&u.Chapa, &u.Nome, &u.Sexo, &u.Cpf, &u.Funcao, &u.Nascimento, &u.Email,
			&u.Admissao, &u.Carteiratrab, &u.Seriecarttrab, &u.Pispasep, &u.Telefone, &u.Idcompany)

		if err != nil {
			return nil, fmt.Errorf("erro ao fazer Scan dos dados: %w", err)
		}
		usuarios = append(usuarios, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração das linhas: %w", err)
	}

	log.Printf("Foram encontrados %d registros na view TANGERINO_USERS.\n", len(usuarios))
	return usuarios, nil
}

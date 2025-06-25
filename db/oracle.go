package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/godror/godror"
)

const dsn = `user="rm" password="rm2b605" connectString="20.1.10.121:1521/CORP"`

func NewOracleConnection() (*sql.DB, error) {
	db, err := sql.Open("godror", dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar no Oracle: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("falha no ping ao Oracle: %w", err)
	}

	// Ajusta o formato de data para toda a sessão
	if _, err := db.Exec(`ALTER SESSION SET NLS_DATE_FORMAT = 'DD/MM/YYYY'`); err != nil {
		log.Printf("Aviso: falha ao alterar NLS_DATE_FORMAT: %v", err)
	} else {
		log.Println("SESSION NLS_DATE_FORMAT ajustado para DD/MM/YYYY")
	}

	log.Println("Conexão com Oracle estabelecida com sucesso!")
	return db, nil
}

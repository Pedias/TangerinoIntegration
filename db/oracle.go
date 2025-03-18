package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/godror/godror"
)

// NewOracleConnection abre e retorna uma conexão com o banco Oracle
func NewOracleConnection() (*sql.DB, error) {
	user := os.Getenv("ORACLE_USER")
	password := os.Getenv("ORACLE_PASSWORD")
	connectString := os.Getenv("ORACLE_CONNECT_STRING")

	if user == "" || password == "" || connectString == "" {
		return nil, fmt.Errorf("variáveis de ambiente para conexão Oracle não foram definidas")
	}

	dsn := fmt.Sprintf(`user="%s" password="%s" connectString="%s"`, user, password, connectString)
	db, err := sql.Open("godror", dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar no Oracle: %w", err)
	}

	// Valida a conexão rapidamente
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("falha no ping ao Oracle: %w", err)
	}

	log.Println("Conexão com Oracle estabelecida com sucesso!")
	return db, nil
}

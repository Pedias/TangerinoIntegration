package main

import (
	"fmt"
	"log"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"TangerinoIntegration/api"
	"TangerinoIntegration/db"

	"github.com/joho/godotenv"
)

func init() {
	// Carrega as variáveis do .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Nenhum arquivo .env encontrado. Variáveis de ambiente devem ser definidas manualmente.")
	}
}

// ParseDDMMYYYYToMillis converte uma data no formato "DD/MM/YYYY" para timestamp em milissegundos.
func ParseDDMMYYYYToMillis(dateStr string) (string, error) {
	layout := "02/01/2006"
	t, err := time.Parse(layout, dateStr)
	if err != nil {
		return "", err
	}
	ms := t.UnixMilli()
	return strconv.FormatInt(ms, 10), nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print("Informe o modo (--insert ou --update): ")
		var mode string
		fmt.Scanln(&mode)
		os.Args = append(os.Args, mode)
	}
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go --insert    // Para inserir novos registros")
		fmt.Println("   ou: go run main.go --update   // Para atualizar registros existentes")
		return
	}

	mode := os.Args[1]

	// 1. Conexão Oracle
	conn, err := db.NewOracleConnection()
	if err != nil {
		log.Fatalf("Erro ao criar conexão: %v", err)
	}
	defer conn.Close()

	// 2. Busca usuários da view TANGERINO_USERS
	users, err := db.GetTangerinoUsers(conn)
	if err != nil {
		log.Fatalf("Erro ao buscar usuários: %v", err)
	}

	// 3. Para cada usuário, converte datas e monta payload
	for _, u := range users {
		admissionDateStr, err := ParseDDMMYYYYToMillis(u.Admissao)
		if err != nil {
			log.Printf("Erro ao parsear data de admissão (CHAPA=%s): %v\n", u.Chapa, err)
			continue
		}
		effectiveDateStr := strconv.FormatInt(time.Now().UnixMilli(), 10)
		birthDateStr, err := ParseDDMMYYYYToMillis(u.Nascimento)
		if err != nil {
			log.Printf("Erro ao parsear data de nascimento (CHAPA=%s): %v\n", u.Chapa, err)
			continue
		}
		//tratamento de genero
		var gender string
		switch strings.ToUpper(u.Sexo) {
		case "M":
			gender = "MASCULINO"
		case "F":
			gender = "FEMININO"
		default:
			gender = "MASCULINO"
		}

		// Validação do e-mail
		var email string
		if strings.TrimSpace(u.Email) != "" {
			if _, err := mail.ParseAddress(u.Email); err != nil {
				log.Printf("Email inválido para usuário CHAPA=%s: %s. Definindo email como vazio.", u.Chapa, u.Email)
				email = ""
			} else {
				email = u.Email
			}
		}

		// Define se o usuário é estagiário com base no campo Funcao da view
		intern := false
		funcao := strings.TrimSpace(strings.ToUpper(u.Funcao))
		if funcao == "ESTAGIÁRIO(A)" {
			intern = true
		}

		// Monta o payload utilizando todos os campos
		payload := api.TangerinoEmployeePayload{
			Name:          u.Nome,
			Cpf:           u.Cpf,
			AdmissionDate: admissionDateStr,
			EffectiveDate: effectiveDateStr,
			Email:         email,
			ExternalId:    u.Chapa, // Usado para identificar o funcionário no update
			BirthDate:     birthDateStr,
			Carteiratrab:  u.Carteiratrab,
			Seriecarttrab: u.Seriecarttrab,
			Pispasep:      u.Pispasep,
			Telefone:      u.Telefone,
			Cargo:         u.Funcao, // Mapeando o campo função para o campo "jobRoleDescription" da API
			Gender:        gender,
			Intern:        intern,
		}

		// Seleciona qual endpoint chamar com base no argumento de linha de comando
		switch mode {
		case "--insert":
			err = api.PostEmployeeToTangerino(payload)
			if err != nil {
				log.Printf("Falha ao inserir usuário CHAPA=%s: %v\n", u.Chapa, err)
			} else {
				log.Printf("Inserção de colaborador CHAPA=%s bem-sucedida.\n", u.Chapa)
			}
		case "--update":
			err = api.PostEmployeeToTangerinoUpdate(payload)
			if err != nil {
				log.Printf("Falha ao atualizar usuário CHAPA=%s: %v\n", u.Chapa, err)
			} else {
				log.Printf("Atualização de colaborador CHAPA=%s bem-sucedida.\n", u.Chapa)
			}
		default:
			fmt.Println("Opção inválida. Use --insert ou --update")
			return
		}
	}

	fmt.Println("Finalizado.")
}

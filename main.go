package main

import (
	"fmt"
	"log"
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"TangerinoIntegration/api"
	"TangerinoIntegration/company"
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

// ParseDDMMYYYYToMillis converte uma data no formato "DD/MM/YYYY" para timestamp em milissegundos, interpretando a data como UTC.
func ParseDDMMYYYYToMillis(dateStr string) (string, error) {
	layout := "02/01/2006"
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return "", fmt.Errorf("erro ao carregar fuso horário: %w", err)
	}

	t, err := time.ParseInLocation(layout, dateStr, loc)
	if err != nil {
		return "", err
	}

	ms := t.UnixMilli()
	return strconv.FormatInt(ms, 10), nil
}
func RemoveMascaraTelefone(telefone string) string {
	re := regexp.MustCompile(`\D`) // \D pega tudo que não é dígito
	return re.ReplaceAllString(telefone, "")
}

func main() {
	// Se não houver argumentos, solicita ao usuário
	if len(os.Args) < 2 {
		fmt.Print("Informe o modo (--insert, --update, --companyupload ou --workplaceupload): ")
		var mode string
		fmt.Scanln(&mode)
		os.Args = append(os.Args, mode)
	}
	if len(os.Args) < 2 {
		fmt.Println("Uso:")
		fmt.Println("  go run main.go --insert        // Para inserir novos registros de funcionários")
		fmt.Println("  go run main.go --update        // Para atualizar registros de funcionários")
		fmt.Println("  go run main.go --companyupload // Para enviar dados de filiais")
		fmt.Println("  go run main.go --workplaceupload // Para enviar dados de setores")
		return
	}

	mode := os.Args[1]

	// Conecta ao Oracle
	conn, err := db.NewOracleConnection()
	if err != nil {
		log.Fatalf("Erro ao criar conexão: %v", err)
	}
	defer conn.Close()

	// Se o modo for companyupload, executa a rotina de envio de filiais
	if mode == "--companyupload" {
		companies, err := db.GetTangerinoCompanies(conn)
		if err != nil {
			log.Fatalf("Erro ao buscar dados de filiais: %v", err)
		}

		for _, c := range companies {
			payload := company.CompanyPayload{
				Cnpj:            c.Cnpj,
				DescriptionName: c.RazaoSocial,
				ExternalId:      c.CodFilial,
				FantasyName:     c.NomeFantasia,
				SocialReason:    c.RazaoSocial,
			}

			err = company.PostCompanyToTangerino(payload)
			if err != nil {
				log.Printf("Falha ao enviar filial CODFILIAL=%s: %v\n", c.CodFilial, err)
			} else {
				log.Printf("Filial CODFILIAL=%s enviada com sucesso.\n", c.CodFilial)
			}
		}
		fmt.Println("Envio de filiais finalizado.")
		return
	}

	// Se o modo for workplaceupload, executa a rotina de envio de setores
	if mode == "--workplaceupload" {
		workplaces, err := db.GetTangerinoWorkplaces(conn)
		if err != nil {
			log.Fatalf("Erro ao buscar dados de setores: %v", err)
		}

		for _, w := range workplaces {
			payload := api.TangerinoWorkplacePayload{
				ExternalId: w.CodSetor,
				Name:       w.NomeSetor,
			}

			err = api.PostWorkplaceToTangerino(payload)
			if err != nil {
				log.Printf("Falha ao enviar setor CODSETOR=%s: %v\n", w.CodSetor, err)
			} else {
				log.Printf("Setor CODSETOR=%s enviado com sucesso.\n", w.CodSetor)
			}
		}
		fmt.Println("Envio de setores finalizado.")
		return
	}

	// Se não for companyupload nem workplaceupload, processa os dados de funcionários
	users, err := db.GetTangerinoUsers(conn)
	if err != nil {
		log.Fatalf("Erro ao buscar usuários: %v", err)
	}

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

		// Tratamento de gênero
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
			ExternalId:    u.Chapa,
			BirthDate:     birthDateStr,
			Carteiratrab:  u.Carteiratrab,
			Seriecarttrab: u.Seriecarttrab,
			Pispasep:      u.Pispasep,
			Telefone:      RemoveMascaraTelefone(u.Telefone),
			Cargo:         u.Funcao,
			Gender:        gender,
			Intern:        intern,
			Company:       u.Idcompany,
			Workplace:     u.Setor,
		}
		/*if mode == "--insert" {
			payload.Company = strings.TrimSpace(u.Codfilial)
		}*/
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
			fmt.Println("Opção inválida. Use --insert, --update, --companyupload ou --workplaceupload")
			return
		}
	}

	fmt.Println("Finalizado.")
}

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
	_ = godotenv.Load()
}

// ParseDDMMYYYYToMillis converte "DD/MM/YYYY" para UnixMilli em America/Sao_Paulo.
func ParseDDMMYYYYToMillis(dateStr string) (string, error) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return "", err
	}
	t, err := time.ParseInLocation("02/01/2006", dateStr, loc)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(t.UnixMilli(), 10), nil
}

// RemoveMascaraTelefone deixa apenas dígitos.
func RemoveMascaraTelefone(telefone string) string {
	re := regexp.MustCompile(`\D`)
	return re.ReplaceAllString(telefone, "")
}

func main() {
	// 1) Leitura do modo
	if len(os.Args) < 2 {
		fmt.Print("Modo (--insert, --update, --dismiss, --companyupload, --workplaceupload): ")
		var m string
		fmt.Scanln(&m)
		os.Args = append(os.Args, m)
	}
	if len(os.Args) < 2 {
		log.Fatal("Uso: --insert | --update | --dismiss | --companyupload | --workplaceupload")
	}
	mode := os.Args[1]

	// 2) Conexão com Oracle
	conn, err := db.NewOracleConnection()
	if err != nil {
		log.Fatalf("Erro ao criar conexão Oracle: %v", err)
	}
	defer conn.Close()

	// 3) Company upload
	if mode == "--companyupload" {
		companies, err := db.GetTangerinoCompanies(conn)
		if err != nil {
			log.Fatalf("Erro ao buscar filiais: %v", err)
		}
		for _, c := range companies {
			payload := company.CompanyPayload{
				Cnpj:            c.Cnpj,
				DescriptionName: c.RazaoSocial,
				ExternalId:      c.CodFilial,
				FantasyName:     c.NomeFantasia,
				SocialReason:    c.RazaoSocial,
			}
			if err := company.PostCompanyToTangerino(payload); err != nil {
				log.Printf("Filial %s erro: %v", c.CodFilial, err)
			} else {
				log.Printf("Filial %s enviada.", c.CodFilial)
			}
		}
		return
	}

	// 4) Workplace upload
	if mode == "--workplaceupload" {
		workplaces, err := db.GetTangerinoWorkplaces(conn)
		if err != nil {
			log.Fatalf("Erro ao buscar setores: %v", err)
		}
		for _, w := range workplaces {
			payload := api.TangerinoWorkplacePayload{
				ExternalId: w.CodSetor,
				Name:       w.NomeSetor,
			}
			if err := api.PostWorkplaceToTangerino(payload); err != nil {
				log.Printf("Setor %s erro: %v", w.CodSetor, err)
			} else {
				log.Printf("Setor %s enviado.", w.CodSetor)
			}
		}
		return
	}

	// 5) Busca usuários
	users, err := db.GetTangerinoUsers(conn)
	if err != nil {
		log.Fatalf("Erro ao buscar usuários: %v", err)
	}

	// 6) Demissão
	if mode == "--dismiss" {
		for _, u := range users {
			dem := strings.TrimSpace(u.Demissao)
			if dem == "" {
				continue
			}
			msStr, err := ParseDDMMYYYYToMillis(dem)
			if err != nil {
				log.Printf("Data demissão inválida CHAPA=%s: %s (%v)", u.Chapa, dem, err)
				continue
			}
			demTs, err := strconv.ParseInt(msStr, 10, 64)
			if err != nil {
				log.Printf("Erro convertendo demissão CHAPA=%s: %v", u.Chapa, err)
				continue
			}
			payload := api.DismissEmployeePayload{
				ExternalId:      u.Chapa,
				ResignationDate: demTs,
			}
			if err := api.DismissEmployee(payload); err != nil {
				log.Printf("Falha ao demitir CHAPA=%s: %v", u.Chapa, err)
			} else {
				log.Printf("Demissão CHAPA=%s enviada com sucesso.", u.Chapa)
			}
		}
		return
	}

	// 7) Insert / Update
	// Data mínima de efetivação (rollout Solides date)
	const rolloutDate = "01/04/2025"
	rolloutMsStr, err := ParseDDMMYYYYToMillis(rolloutDate)
	if err != nil {
		log.Fatalf("Erro parsing rollout date (%s): %v", rolloutDate, err)
	}
	rolloutTs, err := strconv.ParseInt(rolloutMsStr, 10, 64)
	if err != nil {
		log.Fatalf("Erro convertendo rolloutTs: %v", err)
	}

	for _, u := range users {
		// Pula ex-funcionários demitidos no insert conforme CodSituacao
		if mode == "--insert" && u.CodSituacao == "D" {
			continue
		}

		// Parse admissão
		admissionMsStr, err := ParseDDMMYYYYToMillis(u.Admissao)
		if err != nil {
			log.Printf("Admissão inválida CHAPA=%s: %v", u.Chapa, err)
			continue
		}
		admissionTs, err := strconv.ParseInt(admissionMsStr, 10, 64)
		if err != nil {
			log.Printf("Erro convertendo admissionTs CHAPA=%s: %v", u.Chapa, err)
			continue
		}
		// Define effective date mínimo
		effectiveTsInt := admissionTs
		if admissionTs < rolloutTs {
			effectiveTsInt = rolloutTs
		}
		effectiveMsStr := strconv.FormatInt(effectiveTsInt, 10)

		// Parse nascimento
		birthMsStr, err := ParseDDMMYYYYToMillis(u.Nascimento)
		if err != nil {
			log.Printf("Nascimento inválido CHAPA=%s: %v", u.Chapa, err)
			continue
		}

		// Gênero
		gender := "MASCULINO"
		if strings.EqualFold(u.Sexo, "F") {
			gender = "FEMININO"
		}

		// Email
		email := u.Email
		if _, err := mail.ParseAddress(email); err != nil {
			log.Printf("Email inválido CHAPA=%s: %q; omitindo", u.Chapa, email)
			email = ""
		}

		// Intern
		intern := false
		if strings.Contains(strings.ToUpper(u.Funcao), "ESTAGIÁRIO") {
			intern = true
		}

		// Monta payload
		payload := api.TangerinoEmployeePayload{
			Name:                u.Nome,
			Cpf:                 u.Cpf,
			AdmissionDate:       admissionMsStr,
			EffectiveDate:       effectiveMsStr,
			ExternalId:          u.Chapa,
			Email:               email,
			BirthDate:           birthMsStr,
			Carteiratrab:        u.Carteiratrab,
			Seriecarttrab:       u.Seriecarttrab,
			Pispasep:            u.Pispasep,
			Telefone:            RemoveMascaraTelefone(u.Telefone),
			Cargo:               u.Funcao,
			Gender:              gender,
			Intern:              intern,
			Company:             strconv.Itoa(u.Idcompany),
			Chapa:               u.Chapa,
			WorkplaceExternalId: u.Setor,
		}

		var errSend error
		if mode == "--insert" {
			errSend = api.PostEmployeeToTangerino(payload)
		} else {
			errSend = api.PostEmployeeToTangerinoUpdate(payload)
		}

		if errSend != nil {
			log.Printf("CHAPA=%s erro: %v", u.Chapa, errSend)
		} else {
			action := map[string]string{"--insert": "Inserção", "--update": "Atualização"}[mode]
			log.Printf("%s CHAPA=%s bem-sucedida.", action, u.Chapa)
		}
	}

	fmt.Println("Finalizado.")
}

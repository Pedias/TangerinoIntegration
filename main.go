package main

import (
	"fmt"
	"io"
	"log"
	"net/mail"
	"os"
	"path/filepath"
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
		log.Printf("Erro ao carregar localização: %v", err)
		return "", err
	}
	t, err := time.ParseInLocation("02/01/2006", dateStr, loc)
	if err != nil {
		log.Printf("ParseDDMMYYYYToMillis falhou para %s: %v", dateStr, err)
		return "", err
	}
	ms := strconv.FormatInt(t.UnixMilli(), 10)
	log.Printf("ParseDDMMYYYYToMillis: entrada %s -> %s", dateStr, ms)
	return ms, nil
}

// RemoveMascaraTelefone deixa apenas dígitos.
func RemoveMascaraTelefone(telefone string) string {
	re := regexp.MustCompile(`\D`)
	clean := re.ReplaceAllString(telefone, "")
	log.Printf("RemoveMascaraTelefone: entrada %s -> %s", telefone, clean)
	return clean
}

func main() {
	log.Println("Aplicação iniciada")

	// 1) Leitura do modo
	if len(os.Args) < 2 {
		fmt.Print("Modo (--insert, --update, --dismiss, --companyupload, --workplaceupload): ")
		var m string
		_, err := fmt.Scanln(&m)
		if err != nil {
			log.Printf("Erro na leitura do modo: %v", err)
			return
		}
		os.Args = append(os.Args, m)
		log.Printf("Modo informado por prompt: %s", m)
	}
	if len(os.Args) < 2 {
		log.Fatal("Uso: --insert | --update | --dismiss | --companyupload | --workplaceupload")
	}
	mode := strings.TrimPrefix(os.Args[1], "--")
	log.Printf("Modo selecionado: %s", mode)

	// 2) Inicializa sistema de logs
	logDir := "LOG"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Erro ao criar diretório de logs: %v", err)
	}
	timestamp := time.Now().Format("02-01-2006-15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.txt", mode, timestamp))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Erro ao criar arquivo de log: %v", err)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Printf("Erro ao fechar arquivo de log: %v", err)
		}
	}()
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	// 3) Conexão com Oracle
	conn, err := db.NewOracleConnection()
	if err != nil {
		log.Fatalf("Erro ao criar conexão Oracle: %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// 4) Upload de empresas
	if mode == "companyupload" {
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
			log.Printf("CompanyPayload: %+v", payload)
			if err := company.PostCompanyToTangerino(payload); err != nil {
				log.Printf("Filial %s erro: %v", c.CodFilial, err)
			} else {
				log.Printf("Filial %s enviada com sucesso", c.CodFilial)
			}
		}
		return
	}

	// 5) Upload de setores
	if mode == "workplaceupload" {
		workplaces, err := db.GetTangerinoWorkplaces(conn)
		if err != nil {
			log.Fatalf("Erro ao buscar setores: %v", err)
		}
		for _, w := range workplaces {
			payload := api.TangerinoWorkplacePayload{
				ExternalId: w.CodSetor,
				Name:       w.NomeSetor,
			}
			log.Printf("WorkplacePayload: %+v", payload)
			if err := api.PostWorkplaceToTangerino(payload); err != nil {
				log.Printf("Setor %s erro: %v", w.CodSetor, err)
			} else {
				log.Printf("Setor %s enviado com sucesso", w.CodSetor)
			}
		}
		return
	}

	// 6) Busca usuários
	users, err := db.GetTangerinoUsers(conn)
	if err != nil {
		log.Fatalf("Erro ao buscar usuários: %v", err)
	}

	// 7) Demissão
	if mode == "dismiss" {
		for _, u := range users {
			dem := strings.TrimSpace(u.Demissao)
			if dem == "" {
				continue
			}
			msStr, _ := ParseDDMMYYYYToMillis(dem)
			demTs, _ := strconv.ParseInt(msStr, 10, 64)
			payload := api.DismissEmployeePayload{
				ExternalId:      u.Chapa,
				ResignationDate: demTs,
			}
			log.Printf("DismissPayload: %+v", payload)
			const maxDismissRetries = 3
			for attempt := 1; attempt <= maxDismissRetries; attempt++ {
				err := api.DismissEmployee(payload)
				if err != nil {
					// se for 5xx, faz retry
					if strings.Contains(err.Error(), "HTTP 5") && attempt < maxDismissRetries {
						log.Printf("Tentativa %d para CHAPA=%s falhou com %v; retry em %d s…",
							attempt, u.Chapa, err, attempt*2)
						time.Sleep(time.Duration(attempt*2) * time.Second)
						continue
					}
					// erro final
					log.Printf("Erro demissão CHAPA=%s após %d tentativas: %v", u.Chapa, attempt, err)
				} else {
					log.Printf("Demissão CHAPA=%s enviada com sucesso (tentativa %d)", u.Chapa, attempt)
				}
				break
			}
		}
		return
	}

	// 8) Insert / Update
	// Computa fixed date e converte para ms
	fixedDate := "02/06/2025"
	fixedMsStr, _ := ParseDDMMYYYYToMillis(fixedDate)
	fixedMsInt, _ := strconv.ParseInt(fixedMsStr, 10, 64)

	for _, u := range users {
		// DEBUG: ver valor lido
		status := strings.TrimSpace(u.CodSituacao)
		log.Printf("DEBUG CodSituacao CHAPA=%s: %q", u.Chapa, status)
		log.Printf("DEBUG Demissao CHAPA=%s: %q", u.Chapa, u.Demissao)

		// Skip de TODOS os demitidos, em insert **e** update
		dem := strings.TrimSpace(u.Demissao)
		if dem != "" {
			log.Printf("Pulando CHAPA=%s pois possui data de demissão (%s)", u.Chapa, dem)
			continue
		}

		// parse data de admissao
		admissionMsStr, err := ParseDDMMYYYYToMillis(u.Admissao)
		if err != nil {
			log.Printf("Admissão inválida CHAPA=%s: %v", u.Chapa, err)
			continue
		}
		admissionMsInt, _ := strconv.ParseInt(admissionMsStr, 10, 64)
		log.Printf("AdmissionDate (Unix ms) CHAPA=%s: %s", u.Chapa, admissionMsStr)

		// determina EffectiveDate: se admission > fixedDate, usa admission; senão fixedDate
		effectiveMsStr := fixedMsStr
		if admissionMsInt > fixedMsInt {
			effectiveMsStr = admissionMsStr
		}
		log.Printf("EffectiveDate usada CHAPA=%s: %s", u.Chapa, effectiveMsStr)

		// demais campos
		birthMsStr, _ := ParseDDMMYYYYToMillis(u.Nascimento)
		gender := "MASCULINO"
		if strings.EqualFold(u.Sexo, "F") {
			gender = "FEMININO"
		}
		email := strings.TrimSpace(u.Email)
		if _, err := mail.ParseAddress(email); err != nil {
			email = ""
		}
		intern := strings.Contains(strings.ToUpper(u.Funcao), "ESTAGIÁRIO")

		// monta payload completo
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
		log.Printf("Payload preparado para CHAPA=%s: %+v", u.Chapa, payload)

		var errSend error
		if mode == "insert" {
			errSend = api.PostEmployeeToTangerino(payload)
			log.Printf("Chamando PostEmployeeToTangerino para CHAPA=%s", u.Chapa)
		} else {
			errSend = api.PostEmployeeToTangerinoUpdate(payload)
			log.Printf("Chamando PostEmployeeToTangerinoUpdate para CHAPA=%s", u.Chapa)
		}

		if errSend != nil {
			log.Printf("CHAPA=%s erro: %v", u.Chapa, errSend)
		} else {
			action := map[string]string{"insert": "Inserção", "update": "Atualização"}[mode]
			log.Printf("%s CHAPA=%s bem-sucedida", action, u.Chapa)
			log.Printf("==================================================================================================================================================================================")
		}
	}

	log.Println("Finalizado.")
}

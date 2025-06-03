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
	log.Println("Verificando parâmetros de linha de comando")
	if len(os.Args) < 2 {
		fmt.Print("Modo (--insert, --update, --dismiss): ") //--companyupload | --workplaceupload
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

	// 2) Inicializa log para arquivo e console com nome baseado no modo e data
	log.Println("Inicializando sistema de logs")
	logDir := "LOG"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Erro ao criar diretório de logs: %v", err)
	}
	timestamp := time.Now().Format("02-01-2006-15-04")
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
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.Println("Logs direcionados para console e arquivo")

	// 3) Conexão com Oracle
	log.Println("Conectando ao Oracle...")
	conn, err := db.NewOracleConnection()
	if err != nil {
		log.Fatalf("Erro ao criar conexão Oracle: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Erro ao fechar conexão Oracle: %v", err)
		} else {
			log.Println("Conexão Oracle encerrada")
		}
	}()
	log.Println("Conexão Oracle estabelecida com sucesso")

	// 4) Company upload
	if mode == "companyupload" {
		log.Println("Iniciando upload de empresas")
		companies, err := db.GetTangerinoCompanies(conn)
		if err != nil {
			log.Fatalf("Erro ao buscar filiais: %v", err)
		}
		log.Printf("Total de empresas a enviar: %d", len(companies))
		for _, c := range companies {
			log.Printf("Enviando filial: %s", c.CodFilial)
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
				log.Printf("Filial %s enviada com sucesso", c.CodFilial)
			}
		}
		log.Println("Upload de empresas concluído")
		return
	}

	// 5) Workplace upload
	if mode == "workplaceupload" {
		log.Println("Iniciando upload de setores")
		workplaces, err := db.GetTangerinoWorkplaces(conn)
		if err != nil {
			log.Fatalf("Erro ao buscar setores: %v", err)
		}
		log.Printf("Total de setores a enviar: %d", len(workplaces))
		for _, w := range workplaces {
			log.Printf("Enviando setor: %s", w.CodSetor)
			payload := api.TangerinoWorkplacePayload{
				ExternalId: w.CodSetor,
				Name:       w.NomeSetor,
			}
			if err := api.PostWorkplaceToTangerino(payload); err != nil {
				log.Printf("Setor %s erro: %v", w.CodSetor, err)
			} else {
				log.Printf("Setor %s enviado com sucesso", w.CodSetor)
			}
		}
		log.Println("Upload de setores concluído")
		return
	}

	// 6) Busca usuários
	log.Println("Buscando usuários no banco")
	users, err := db.GetTangerinoUsers(conn)
	if err != nil {
		log.Fatalf("Erro ao buscar usuários: %v", err)
	}
	log.Printf("Total de usuários encontrados: %d", len(users))

	// 7) Demissão
	if mode == "dismiss" {
		log.Println("Iniciando demissão de usuários")
		for _, u := range users {
			log.Printf("Processando demissão para CHAPA=%s", u.Chapa)
			dem := strings.TrimSpace(u.Demissao)
			if dem == "" {
				log.Printf("Nenhuma data de demissão para CHAPA=%s, pulando", u.Chapa)
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
			log.Printf("Enviando demissão para CHAPA=%s com timestamp %d", u.Chapa, demTs)
			payload := api.DismissEmployeePayload{
				ExternalId:      u.Chapa,
				ResignationDate: demTs,
			}
			if err := api.DismissEmployee(payload); err != nil {
				log.Printf("Falha ao demitir CHAPA=%s: %v", u.Chapa, err)
			} else {
				log.Printf("Demissão CHAPA=%s enviada com sucesso", u.Chapa)
			}
		}
		log.Println("Processo de demissão finalizado")
		return
	}

	// 8) Insert / Update
	log.Println("Iniciando processamento de insert/update de usuários")
	// Data fixa para EffectiveDate
	fixedDate := "02/06/2025"
	fixedMsStr, err := ParseDDMMYYYYToMillis(fixedDate)
	if err != nil {
		log.Fatalf("Erro parsing fixed effective date (%s): %v", fixedDate, err)
	}
	log.Printf("Fixed EffectiveDate (Unix ms): %s", fixedMsStr)

	for _, u := range users {
		log.Printf("Processando usuário CHAPA=%s", u.Chapa)
		if mode == "insert" && strings.TrimSpace(u.CodSituacao) == "D" {
			log.Printf("Pulando CHAPA=%s pois CodSituacao=D", u.Chapa)
			continue
		}

		admissionMsStr, err := ParseDDMMYYYYToMillis(u.Admissao)
		if err != nil {
			log.Printf("Admissão inválida CHAPA=%s: %v", u.Chapa, err)
			continue
		}
		log.Printf("AdmissionDate (Unix ms) para CHAPA=%s: %s", u.Chapa, admissionMsStr)

		birthMsStr, err := ParseDDMMYYYYToMillis(u.Nascimento)
		if err != nil {
			log.Printf("Nascimento inválido CHAPA=%s: %v", u.Chapa, err)
			continue
		}
		log.Printf("BirthDate (Unix ms) para CHAPA=%s: %s", u.Chapa, birthMsStr)

		gender := "MASCULINO"
		if strings.EqualFold(u.Sexo, "F") {
			gender = "FEMININO"
		}
		log.Printf("Gênero para CHAPA=%s: %s", u.Chapa, gender)

		email := strings.TrimSpace(u.Email)
		if _, err := mail.ParseAddress(email); err != nil {
			log.Printf("Email inválido CHAPA=%s: %q; omitindo", u.Chapa, email)
			email = ""
		}
		log.Printf("Email válido (ou vazio) para CHAPA=%s: %q", u.Chapa, email)

		intern := false
		if strings.Contains(strings.ToUpper(u.Funcao), "ESTAGIÁRIO") {
			intern = true
		}
		log.Printf("Intern para CHAPA=%s: %v", u.Chapa, intern)

		payload := api.TangerinoEmployeePayload{
			Name:                u.Nome,
			Cpf:                 u.Cpf,
			AdmissionDate:       admissionMsStr,
			EffectiveDate:       fixedMsStr,
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
		}
	}

	log.Println("Finalizado.")
}

package models

// TangerinoCompany representa os dados de uma filial conforme a view TANGERINO_COMPANY
type TangerinoCompany struct {
	CodFilial    string // Campo usado para externalId e id
	RazaoSocial  string // Social Reason
	NomeFantasia string // Fantasy Name
	Cnpj         string // CNPJ da filial
}

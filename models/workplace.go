package models

// TangerinoWorkplace representa os dados de um setor conforme a view TANGERINO_SETOR
type TangerinoWorkplace struct {
	CodSetor  string // Código do setor (será usado como externalId)
	NomeSetor string // Nome do setor
}

package models

// TangerinoUser representa os usuários do Tangerino no banco de dados.
type TangerinoUser struct {
	Chapa         string
	Nome          string
	Sexo          string // vem como 'M'/'F' da view
	Cpf           string
	Funcao        string
	Nascimento    string // vem como "DD/MM/YYYY" da view
	Email         string
	Admissao      string // vem como "DD/MM/YYYY" da view
	Carteiratrab  string
	Seriecarttrab string
	Pispasep      string
	Telefone      string
	Idcompany     int
}

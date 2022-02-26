package model

//GetContracts
func GetContract() (int64, error) {
	row := db.QueryRow("select block from contract")
	block := int64(0)
	err := row.Scan(&block)
	return block, err
}

func UpdateContract(block int64) error {
	_, err := db.Exec("update contract set block=?", block)
	return err
}

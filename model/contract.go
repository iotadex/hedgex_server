package model

//GetContracts
func GetContract(note string) (int64, error) {
	row := db.QueryRow("select block from contract where note=?", note)
	block := int64(0)
	err := row.Scan(&block)
	return block, err
}

func UpdateContract(note string, block int64) error {
	_, err := db.Exec("update contract set block=? where note=?", block, note)
	return err
}

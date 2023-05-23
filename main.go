package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/nft"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"log"
	"net/http"
	"strings"
)

func connectDB() *pgxpool.Pool {
	connString := "postgres://postgres:12345@localhost:5432/postgres?sslmode=disable"
	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	return pool
}

type User struct {
	Address string
	Seed    string
	ID      string
}

type SignInRequest struct {
	ID   string `json:"id"`
	Seed string `json:"seed"`
}

type TransferTONRequest struct {
	SenderAddress    string `json:"sender_address"`
	AccountID        string `json:"account_id"`
	Amount           string `json:"amount"`
	Comment          string `json:"comment"`
	RecipientAddress string `json:"recipient_address"`
}

type GetTransactionsHistoryRequest struct {
	Address string `json:"address"`
}

type CheckBalanceRequest struct {
	Address string `json:"address"`
}

type CheckBalanceResponse struct {
	Balance string `json:"balance"`
}

type AddressesStruct struct {
	Id string `json:"id"`
}

type NFTstruct struct {
	addr string `json:"NFT_addr"`
}

type Adresses struct {
	Address string `json:"address"`
}

func main() {

	router := mux.NewRouter()

	router.HandleFunc("/createSeed", createSeed).Methods("POST")
	router.HandleFunc("/signWithSeed", signWithSeed).Methods("POST")
	router.HandleFunc("/getTransactionsHistory", getTransactionsHistory).Methods("POST")
	router.HandleFunc("/transferTON", transferTON).Methods("POST")
	router.HandleFunc("/checkBalance", checkBalance).Methods("POST")
	router.HandleFunc("/checkAccount", checkAccount).Methods("POST")
	router.HandleFunc("/getAccounts", getAccounts).Methods("POST")
	router.HandleFunc("/getNFT", getNFTdata).Methods("POST")

	http.ListenAndServe(":8080", router)
}

func checkAccount(w http.ResponseWriter, r *http.Request) {

	client := liteclient.NewConnectionPool()

	err := client.AddConnection(context.Background(), "135.181.140.212:13206", "K0t3+IWLOXHYMvMcrGZDPs+pn58a17LFbnXoQkKc2xw=")
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return
	}
	var req CheckBalanceRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = address.ParseAddr(req.Address)

	if err != nil {
		w.WriteHeader(http.StatusLocked)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func createSeed(w http.ResponseWriter, r *http.Request) {

	client := liteclient.NewConnectionPool()

	configUrl := "https://ton-blockchain.github.io/global.config.json"
	err := client.AddConnectionsFromConfigUrl(context.Background(), configUrl)
	if err != nil {
		log.Fatalln(err)
	}
	api := ton.NewAPIClient(client)

	words := wallet.NewSeed()
	addressUser, err := wallet.FromSeed(api, words, wallet.V3)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	addressString := addressUser.Address().String()
	wordString := strings.Join(words, " ")
	uuidId, _ := uuid.NewRandom() // Implement your ID generation logic
	id := uuidId.String()
	db := connectDB()
	defer db.Close()

	_, err = db.Exec(context.Background(), `INSERT INTO users (id, seed, address) VALUES ($1, $2, $3)`, id, wordString, addressString)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user := User{
		ID:      id,
		Address: addressString,
		Seed:    wordString,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func signWithSeed(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	db := connectDB()
	defer db.Close()

	var user User
	client := liteclient.NewConnectionPool()

	configUrl := "https://ton-blockchain.github.io/global.config.json"
	err = client.AddConnectionsFromConfigUrl(context.Background(), configUrl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	api := ton.NewAPIClient(client)
	seedSlice := strings.Split(req.Seed, " ")

	addressUser, err := wallet.FromSeed(api, seedSlice, wallet.V3)
	if err != nil {
		w.WriteHeader(http.StatusLocked)
		return
	}
	addressString := addressUser.Address().String()
	// Запись с таким ID не найдена, создаем новую запись
	user.ID = req.ID
	user.Seed = req.Seed
	user.Address = addressString // Тут предполагается функция, которая возвращает адрес по seed фразе

	_, err = db.Exec(context.Background(), `INSERT INTO users (id, seed, address) VALUES ($1, $2, $3)`, user.ID, user.Seed, user.Address)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func getTransactionsHistory(w http.ResponseWriter, r *http.Request) {
	var req GetTransactionsHistoryRequest

	client := liteclient.NewConnectionPool()

	err := client.AddConnection(context.Background(), "135.181.140.212:13206", "K0t3+IWLOXHYMvMcrGZDPs+pn58a17LFbnXoQkKc2xw=")
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return
	}

	api := ton.NewAPIClient(client)
	ctx := client.StickyContext(context.Background())

	b, err := api.CurrentMasterchainInfo(ctx)
	if err != nil {
		log.Fatalln("get block err:", err.Error())
		return
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userAddress := req.Address
	addr := address.MustParseAddr(userAddress)

	res, err := api.WaitForBlock(b.SeqNo).GetAccount(ctx, b, addr)
	if err != nil {
		log.Fatalln("get account err:", err.Error())
		return
	}

	fmt.Printf("Is active: %v\n", res.IsActive)
	if res.IsActive {
		fmt.Printf("Status: %s\n", res.State.Status)
		fmt.Printf("Balance: %s TON\n", res.State.Balance.TON())
		if res.Data != nil {
			fmt.Printf("Data: %s\n", res.Data.Dump())
		}
	}

	// take last tx info from account info
	lastHash := res.LastTxHash
	lastLt := res.LastTxLT

	list, err := api.ListTransactions(context.Background(), addr, 15, lastLt, lastHash)
	if err != nil {
		// In some cases you can get error:
		// lite server error, code XXX: cannot compute block with specified transaction: lt not in db
		// it means that current lite server does not store older data, you can query one with full history
		log.Printf("send err: %s", err.Error())
		return
	}
	// Получение истории транзакций для указанного адреса
	// Здесь вам необходимо реализовать функцию, которая получает историю транзакций
	// В этом примере, мы просто создаем фиктивный набор байт
	var buf bytes.Buffer

	for i, t := range list {
		if i > 0 {
			buf.WriteString("_")
		}
		buf.WriteString(t.String())
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func getSeedFromDatabase(senderAddress, accountID string) (string, error) {
	db := connectDB()
	defer db.Close()
	var seed string
	err := db.QueryRow(context.Background(), "SELECT seed FROM users WHERE address = $1 AND id = $2", senderAddress, accountID).Scan(&seed)
	if err != nil {
		return "", err
	}
	return seed, nil
}

func transferTON(w http.ResponseWriter, r *http.Request) {
	var req TransferTONRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	senderAddress := req.SenderAddress
	accountID := req.AccountID
	amount := req.Amount
	comment := req.Comment
	recipientAddress := req.RecipientAddress

	seed, err := getSeedFromDatabase(senderAddress, accountID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	client := liteclient.NewConnectionPool()

	err = client.AddConnection(context.Background(), "135.181.140.212:13206", "K0t3+IWLOXHYMvMcrGZDPs+pn58a17LFbnXoQkKc2xw=")
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return
	}
	api := ton.NewAPIClient(client)

	seedSlice := strings.Split(seed, " ")

	walletUser, err := wallet.FromSeed(api, seedSlice, wallet.V3)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	addr := address.MustParseAddr(recipientAddress)

	err = walletUser.Transfer(context.Background(), addr, tlb.MustFromTON(amount), comment)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func checkBalance(w http.ResponseWriter, r *http.Request) {

	client := liteclient.NewConnectionPool()

	err := client.AddConnection(context.Background(), "135.181.140.212:13206", "K0t3+IWLOXHYMvMcrGZDPs+pn58a17LFbnXoQkKc2xw=")
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return
	}
	api := ton.NewAPIClient(client)

	var req CheckBalanceRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	addr := address.MustParseAddr(req.Address)

	ctx := client.StickyContext(context.Background())

	block, err := api.CurrentMasterchainInfo(ctx)
	if err != nil {
		log.Fatalln("get block err:", err.Error())
		return
	}

	res, err := api.WaitForBlock(block.SeqNo).GetAccount(ctx, block, addr)
	if err != nil {
		log.Fatalln("get account err:", err.Error())
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Получение баланса для указанного адреса
	// Здесь вам необходимо реализовать функцию, которая получает баланс
	// В этом примере, мы просто создаем фиктивный баланс
	if res.State == nil {
		resp := CheckBalanceResponse{Balance: "0 TON"}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	} else {
		balance := res.State.Balance.TON()

		resp := CheckBalanceResponse{Balance: balance}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func getAccounts(w http.ResponseWriter, r *http.Request) {
	db := connectDB()
	defer db.Close()
	var req AddressesStruct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rows, err := db.Query(context.Background(), "SELECT address FROM users WHERE id=$1", req.Id)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
	}
	defer rows.Close()
	var users []Adresses
	for rows.Next() {
		var user Adresses
		err = rows.Scan(&user.Address)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
		}
		users = append(users, user)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)

}

func getNFTdata(w http.ResponseWriter, r *http.Request) {
	client := liteclient.NewConnectionPool()

	err := client.AddConnection(context.Background(), "135.181.140.212:13206", "K0t3+IWLOXHYMvMcrGZDPs+pn58a17LFbnXoQkKc2xw=")
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return
	}
	api := ton.NewAPIClient(client)
	var req NFTstruct
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	nftAddr := address.MustParseAddr(req.addr)
	item := nft.NewItemClient(api, nftAddr)

	nftData, err := item.GetNFTData(context.Background())
	if err != nil {
		panic(err)
	}

	// get info about our nft's collection
	collection := nft.NewCollectionClient(api, nftData.CollectionAddress)
	collectionData, err := collection.GetCollectionData(context.Background())
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collectionData)

}

func deleteAccount(w http.ResponseWriter, r *http.Request) {
	// Implement the account deletion logic here
}

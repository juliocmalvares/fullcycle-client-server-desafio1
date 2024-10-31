package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type CotacaoAPIResponse struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

// Estrutura para enviar a resposta ao cliente

func main() {
	http.HandleFunc("/cotacao", handleUSDBRL)
	log.Println("Servidor iniciado na porta 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleUSDBRL(w http.ResponseWriter, r *http.Request) {
	ctxApi, cancelApi := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelApi()
	cotacao, err := BuscaCotacao(ctxApi)
	if err != nil {
		http.Error(w, "Erro ao obter cotação", http.StatusInternalServerError)
		log.Println("Erro ao obter cotação:", err)
		return
	}
	log.Println("Cotação do dólar:", cotacao.USDBRL.Bid)

	ctxDB, cancelDB := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelDB()

	// Inserir a cotação no banco de dados
	err = PersistCotacao(ctxDB, *cotacao)
	if err != nil {
		http.Error(w, "Erro ao salvar cotação", http.StatusInternalServerError)
		log.Println("Erro ao salvar cotação:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cotacao.USDBRL)
}

func BuscaCotacao(ctx context.Context) (*CotacaoAPIResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Timeout ao chamar a API de cotação do dólar")
		}
		return nil, err
	}
	defer resp.Body.Close()
	var apiResponse CotacaoAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	if err != nil {
		return nil, err
	}

	return &apiResponse, nil
}

func PersistCotacao(ctx context.Context, cotacao CotacaoAPIResponse) error {
	db, err := sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		return err
	}
	defer db.Close()
	createTable := `
		CREATE TABLE IF NOT EXISTS cotacoes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL,
			codein TEXT NOT NULL,
			name TEXT NOT NULL,
			high TEXT NOT NULL,
			low TEXT NOT NULL,
			varBid TEXT NOT NULL,
			pctChange TEXT NOT NULL,
			bid TEXT NOT NULL,
			ask TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			create_date TEXT NOT NULL
		);`
	_, err = db.ExecContext(ctx, createTable)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Timeout ao criar a tabela no banco de dados")
		}
		return err
	}

	// Inserir a cotação no banco de dados
	insert := `
		INSERT INTO cotacoes (
			code, codein, name, high, low, varBid, pctChange, bid, ask, timestamp, create_date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`

	_, err = db.ExecContext(ctx, insert,
		cotacao.USDBRL.Code,
		cotacao.USDBRL.Codein,
		cotacao.USDBRL.Name,
		cotacao.USDBRL.High,
		cotacao.USDBRL.Low,
		cotacao.USDBRL.VarBid,
		cotacao.USDBRL.PctChange,
		cotacao.USDBRL.Bid,
		cotacao.USDBRL.Ask,
		cotacao.USDBRL.Timestamp,
		cotacao.USDBRL.CreateDate,
	)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Timeout ao inserir a cotação no banco de dados")
		}
		return err
	}

	return nil
}

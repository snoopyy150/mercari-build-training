package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Item struct {
	ID        string `json:"id"`                   // 商品のID
	Name      string `json:"name"`                 // 商品の名前
	Category  string `json:"category"`             // 商品のカテゴリ
	ImageName string `json:"image_name,omitempty"` // 画像ファイル名
}

type Items struct {
	Items []Item `json:"items"`
}

func main() {
	r := mux.NewRouter()

	// 既存のエンドポイント
	r.HandleFunc("/items", itemsHandler).Methods("POST", "GET")

	// 新しいエンドポイント: 商品の詳細を取得
	r.HandleFunc("/items/{item_id}", getItemHandler).Methods("GET")
	r.HandleFunc("/search", searchItemsHandler).Methods("GET")

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["item_id"]

	// items.json ファイルから商品情報を読み込む
	var items Items
	data, err := ioutil.ReadFile("items.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.Unmarshal(data, &items)

	// itemID に一致する商品を検索
	for _, item := range items.Items {
		if item.ID == itemID { // ID フィールドを Item 構造体に追加する必要があります
			json.NewEncoder(w).Encode(item)
			return
		}
	}

	// 商品が見つからない場合は 404 Not Found を返す
	http.NotFound(w, r)
}

func itemsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Multipart Formのパース
		err := r.ParseMultipartForm(10 << 20) // 最大10MB
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// JSONデータの処理
		var item Item
		item.Name = r.FormValue("name")
		item.Category = r.FormValue("category")
		// 商品情報にIDを割り当てる（ここでは現在のタイムスタンプを使用）
		item.ID = strconv.FormatInt(time.Now().UnixNano(), 10)

		// 画像の処理
		file, header, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Image is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// ハッシュ値を計算する
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			http.Error(w, "Failed to hash the image", http.StatusInternalServerError)
			return
		}
		hashedFilename := fmt.Sprintf("%x", hash.Sum(nil)) + filepath.Ext(header.Filename)
		item.ImageName = hashedFilename // 商品情報に画像ファイル名を追加

		// ファイルポインタをリセットする
		file.Seek(0, io.SeekStart)

		// ハッシュ化されたファイル名で画像を保存する
		dst, err := os.Create(filepath.Join("images", hashedFilename))
		if err != nil {
			http.Error(w, "Failed to save the image", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to save the image", http.StatusInternalServerError)
			return
		}

		// 商品情報をJSONファイルに保存する処理
		var items Items
		data, err := ioutil.ReadFile("items.json")
		if err == nil {
			json.Unmarshal(data, &items)
		}
		items.Items = append(items.Items, item)
		updatedData, err := json.Marshal(items)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ioutil.WriteFile("items.json", updatedData, 0644)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "item received: " + item.Name})
	} else if r.Method == "GET" {
		// 商品一覧を返す処理
		data, err := ioutil.ReadFile("items.json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
func searchItemsHandler(w http.ResponseWriter, r *http.Request) {
	// クエリパラメータから検索キーワードを取得
	keyword := r.URL.Query().Get("keyword")

	if keyword == "" {
		http.Error(w, "Keyword parameter is missing", http.StatusBadRequest)
		return
	}

	// データベースからキーワードを含む商品を検索
	// 以下は擬似コードです。実際のデータベース操作には適切な処理を行ってください。
	var items []Item // Item は商品を表す構造体
	// ...データベース検索処理...
	// 例: SELECT * FROM items WHERE name LIKE '%keyword%'

	// 検索結果を JSON でレスポンス
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]Item{"items": items})
}

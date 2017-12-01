package requesthandlers

import (
	"github.com/Narazaka/shiorigo"
	"log"
	"math/rand"
	"time"
)

/* main パッケージから適宜 OnLoad(), OnUnload(), OnRequest() が呼ばれます。
 *
 * OnRequest() が呼ばれると、リクエストのイベント ID に応じて Handlers というハンドラテーブルから対応したハンドラを探して実行します。
 * ハンドラは OnLoad() が呼ばれた時に登録しておきます。
 * 各ハンドラではリクエストを基にレスポンスを作成して返します。
 * 特に返す内容が無かったり、リクエストを無視するときは 204 No Content を返します。
 * これにより「正常に終了したが返すものはない」という意図を伝えることができます。
 *
 * 他に、レスポンスの作成が簡単にできるような機能も定義しています。
 *
 * 例えば、Talks 型でトークのリストを定義してその中からランダムにひとつ選んで返すことができます。
 * このとき、各トークはバックスラッシュをエスケープした Sakura Script です。
 *
 * 各 Response〜 系の関数は特定のレスポンスを返すためのレスポンスビルダです。
 * さらに CreateGetHandlerOf() を使うと GET イベントの時に特定の値を返すハンドラを作成できます。
 *
 * ランダムトークは OnSecondChange イベントの度にカウンタを増やし、閾値を超えたら 1/10 の確率でトークを返すという実装になっています。
 * 閾値を超えたがトークを返さない場合に、乱数のシードを変えています。
 * これにより繰り返しがよりランダムに起こることを期待しています。
 *
 * なお、このパッケージでは擬似乱数発生に rand パッケージを使っています。
 * この乱数のシードの更新は、上述の OnSecondChange イベントと OnLoad() が呼ばれた時に ResetRNG() を呼び出して行っています。
 *
 * このパッケージではトークを文字リテラルでコード中に埋め込んでいますが、TOML や Lua などの形で外部にデータを移すとゴースト開発が楽になるでしょう。
 * また、NOTIFY リクエストなどで得た情報を変数に記録しておき、text/template でトークに埋め込むことも可能です。
 * 単語をその種類ごとに分けた辞書を map[string][]string で作り、ある種類の単語を辞書からランダムに取ってくるのは、トークの多様性を増すための古典的な方法です。
 *
 * イベントの種類など SHIORI/3.0 の仕様については「http://ssp.shillest.net/ukadoc/manual/index.html」を見てください。
 */

type RequestHandler func(shiori.Request) (shiori.Response, error)

var (
	Handlers            = map[string]RequestHandler{}
	SecondsFromLastTalk = 0
	TalkFrequency       = 15
	Info                = map[string]string{
		"version":   "1.00",
		"name":      "gohst",
		"craftman":  "kurousada",
		"craftmanw": "kurousada",
	}
)

func ResponseOK(value string) shiori.Response {
	res := shiori.Response{Protocol: shiori.SHIORI, Version: "3.0", Code: 200, Headers: shiori.ResponseHeaders{}}
	if value != "" {
		res.Headers["Value"] = value
	}
	return res
}

func ResponseNoContent() shiori.Response {
	return shiori.Response{Protocol: shiori.SHIORI, Version: "3.0", Code: 204, Headers: shiori.ResponseHeaders{}}
}

func ResponseBadRequest() shiori.Response {
	return shiori.Response{Protocol: shiori.SHIORI, Version: "3.0", Code: 400, Headers: shiori.ResponseHeaders{}}
}

func ResponseInternalServerError() shiori.Response {
	return shiori.Response{Protocol: shiori.SHIORI, Version: "3.0", Code: 500, Headers: shiori.ResponseHeaders{}}
}

type Talks []string

func (values Talks) OneOf() string {
	list := []string{}
	for _, value := range values {
		if value != "" {
			list = append(list, value)
		}
	}
	length := len(list)
	if length <= 0 {
		return ""
	}
	i := rand.Intn(length)
	return list[i]
}

func ResponseOneOf(values Talks) shiori.Response {
	v := values.OneOf()
	if v != "" {
		return ResponseOK(v)
	}
	return ResponseNoContent()
}

func CreateGetHandlerOf(value string) RequestHandler {
	return func(req shiori.Request) (shiori.Response, error) {
		if req.Method == shiori.GET && value != "" {
			return ResponseOK(value), nil
		}
		return ResponseNoContent(), nil
	}
}

func ResetRNG() {
	rand.Seed(time.Now().UnixNano())
}

func OnLoad(path string) error {
	ResetRNG()

	// Info の内容を要求するリクエストに応えてその値を返すハンドラを定義します。
	for event, value := range Info {
		Handlers[event] = CreateGetHandlerOf(value)
	}

	Handlers["OnBoot"] = CreateGetHandlerOf("\\1\\s[10]\\0\\s[1]これは Go で栞を作るサンプルだから、\\w2過度な期待はしないよーに。\\e")
	Handlers["OnClose"] = CreateGetHandlerOf("\\1\\s[10]\\0\\s[5]じゃ、\\w2えんいー！\\_w[500]\\e")
	Handlers["OnSecondChange"] = func(req shiori.Request) (shiori.Response, error) {
		var err error
		SecondsFromLastTalk += 1
		if SecondsFromLastTalk >= TalkFrequency { // 一定時間経ったらランダムトーク
			if r := rand.Intn(10); r <= 0 { // ただし 1/10 の確率で何もトークを返さない（よりランダムに見せるため）。
				ResetRNG() // 話せるのにトークを返さなかった時は乱数のシードを再設定する。
				return ResponseNoContent(), nil
			}
			SecondsFromLastTalk = 0
			return ResponseOneOf(Talks{
				"\\1\\s[10]\\0\\s[0]…\\w1…\\w1\\s[3]ﾆﾔﾘ\\e",
				"\\1\\s[10]\\0\\s[4]ランダムトークはまだ少ししかないのよ\\w1…\\w1…\\e",
				"\\1\\s[10]\\0\\s[2]本当に何もしゃべることがないわね\\w1…\\w1…\\e",
			}), err
		}
		return ResponseNoContent(), err
	}
	return nil
}

func OnUnload() error {
	return nil
}

func OnRequest(req shiori.Request) (shiori.Response, error) {
	var err error = nil

	// デフォルトで 204 No Content を返します。
	res := ResponseNoContent()

	// リクエストヘッダがなければ、統一的操作のために初期化しておきます。
	if req.Headers == nil {
		req.Headers = shiori.RequestHeaders{}
	}

	// ID ヘッダにはイベント名が入っています。
	// イベントに対応するハンドラが定義されていれば呼び出します。
	if req.Headers["ID"] != "" {
		event := req.Headers["ID"]
		handler := Handlers[event]
		if handler != nil {
			res, err = handler(req)
			if err != nil {
				// ハンドラ内でエラーが起きたら Internal Server Error を返します。
				// 実は main パッケージの request() 関数内で同じことをしていますので、無駄です。
				log.Printf("[error] OnRequest() handler \"%s\" failed\n%s\n", event, err.Error())
				res = ResponseInternalServerError()
			}
		}
	}
	return res, err
}

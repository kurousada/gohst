package main

/*
   #define UNICODE

   #include <windows.h>
   #include <stdlib.h>
   #include <string.h>
*/
import "C"

import (
	"bytes"
	"fmt"
	"github.com/Narazaka/shiorigo"
	"github.com/kurousada/gohst/internal/readerstream"
	"github.com/kurousada/gohst/internal/requesthandlers"
	"github.com/kurousada/gohst/internal/win32file"
	"log"
	"os"
	"unsafe"
)

// 空だけどコンパイルするのに必要です。
func main() {}

var (
	logFile *win32file.File
)

/* main パッケージは以下の関数をエクスポートしています。
 *
 *  extern "C" __declspec(dllexport) BOOL __cdecl load(HGLOBAL h, long len);
 *  extern "C" __declspec(dllexport) BOOL __cdecl unload(void);
 *  extern "C" __declspec(dllexport) HGLOBAL __cdecl request(HGLOBAL h, long *len);
 *
 * それぞれの関数では、ログファイルの扱いと internal/requesthandlers パッケージで定義されているリクエストハンドラの呼び出しを行っています。
 *
 */

/* extern "C" __declspec(dllexport) BOOL __cdecl load(HGLOBAL h, long len);
 *
 * h      = DLL のパス（文字列）
 * length = h のサイズ
 *
 * h は GlobalAlloc(GPTR, length) で確保されたメモリ領域へのポインタで、DLL 側で GlobalFree(h) する必要があります。
 *
 * あと、「//export 〜」は「// export 〜」だとダメです。空行もダメ。
 */
//export load
func load(h C.HGLOBAL, length C.long) C.BOOL {
	var err error

	// せっかくサイズ情報があるので C.GoStringN() を使います。
	// ここは C.GoString() でも大丈夫なはず。
	// あ、HGLOBAL は void* なので、char* に相当する *C.char にキャストしています。
	curDir := C.GoStringN((*C.char)(h), (C.int)(length))

	// DLL のパスが入っているメモリを開放します。
	C.GlobalFree(h)

	// Logger の設定をします。
	//
	// log パッケージを使って、"shiori.log" というファイルに書き込みます。
	// ログファイルの場所は DLL と同じ位置（ベースウェアのゴーストフォルダ内「/gohst/ghost/master/shiori.log」）です。
	// ファイルをオープンできなければ標準出力にエラーメッセージを出します。
	//
	// ただ、出力先のファイルの読み書きに os パッケージで開いたファイルを使うと「書き込み権限がない」というエラーが出ます。
	// これはおそらく FAT32 にパーミッションという概念がないことと、私が Wine を使っているせいです。
	// そのため、Win32 API を cgo で直接叩いています（その部分は internal/win32file パッケージにまとめてあります）。
	//
	// それから、Write() するときに Shift_JIS に変換するようにフックを書いています。
	// このフックを登録できる機能は win32file パッケージ独自のもので、os パッケージにはありません（たぶん）。
	//
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetOutput(os.Stdout)
	path := "shioringo.log"
	logFile, err = win32file.OpenFile(path, win32file.O_WRONLY)
	if err != nil {
		fmt.Println(err.Error())
	} else if err == nil && logFile != nil {
		logFile.OnWrite(func(p []byte) ([]byte, error) {
			// Write()時にShift_JISに変換する
			return readerstream.New(bytes.NewReader(p)).ToShiftJIS().ToBytes()
		})
		log.SetOutput(logFile)
	}

	log.Printf("[info] load(\"%s\", %d)\n", curDir, (int)(length))

	// リクエストハンドラを呼びます。
	err = requesthandlers.OnLoad(curDir)
	if err != nil {
		log.Printf("[info] OnLoad() failed\n%s\n", err.Error())
	}

	return C.TRUE
}

/* extern "C" __declspec(dllexport) BOOL __cdecl unload();
 *
 * リクエストハンドラを呼んでログファイルをクローズするだけです。
 */
//export unload
func unload() C.BOOL {
	log.Println("[info] unload()")

	// リクエストハンドラを呼びます
	err := requesthandlers.OnUnload()
	if err != nil {
		log.Printf("[info] OnUnload() failed\n%s\n", err.Error())
	}

	// ログファイルを使っていたら、クローズします。
	if logFile != nil {
		logFile.Close()
	}
	return C.TRUE
}

/* extern "C" __declspec(dllexport) HGLOBAL __cdecl request(HGLOBAL h, long *length);
 *
 * h      = リクエスト（文字列）
 * length = h のサイズ（load() と違ってポインタなので注意）
 *
 * h は GlobalAlloc(GPTR, length) で確保されたメモリ領域へのポインタで、DLL 側で GlobalFree(h) する必要があります。
 *
 * リクエストの文字コードは Charset ヘッダを見るまでわかりませんが、ここでは簡便のため UTF-8 で来ると仮定しています。
 * 伺かベースウェアのデファクトスタンダードである SSP は UTF-8 で送ってくれるので、とりあえず、です。
 * そこら辺もちゃんと処理したいなら Charset ヘッダを見ればリクエストの文字コードがわかります。
 * Shift_JIS から UTF-8 に変換したければ、strings パッケージをインポートして以下を追加します。
 *
 *  req_str = readerstream.New(strings.NewReader(req_str)).FromShiftJIS().String()
 *
 * レスポンスは h とは別に GlobalAlloc(GPTR, n) し、そこに書き込みます。
 * そして書き込んだ長さ n を length に書き込み、レスポンスが入ったメモリ領域へのポインタを返します。
 * こっちの GlobalFree() はベースウェアがしてくれます。
 *
 * レスポンスも簡便のため一律に UTF-8 で返しています。
 * Shift_JIS で返したければ、strings パッケージをインポートして以下を追加します。
 *
 *  res.Headers["Charset"] = "Shift_JIS"
 *  res_str = readerstream.New(strings.NewReader(res_str)).ToShiftJIS().String()
 */
//export request
func request(h C.HGLOBAL, length *C.long) C.HGLOBAL {
	var err error
	var req shiori.Request
	var res shiori.Response

	// リクエストが入っているメモリのサイズを取得します。
	// load() と違い、ポインタなので注意してください。
	req_size := (*length)

	// せっかくサイズ情報があるので C.GoStringN() を使います。
	// ここは C.GoString() でも大丈夫なはず。
	req_str := C.GoStringN((*C.char)(h), (C.int)(req_size))

	// レスポンスが入っているメモリを開放します。
	C.GlobalFree(h)

	log.Printf("[info] request content [%s]\n%s\n", req.Charset(), req_str)

	// リクエストである Go string をパースして shiori.Request にします。
	req, err = shiori.ParseRequest(req_str)
	if err != nil {
		// パースできなかったらダメなリクエスト送ってきてんじゃねえ！と文句を返します（文学的な表現です）。
		log.Printf("[error] shiori.ParseRequest() failed: %s\n", err.Error())
		res = requesthandlers.ResponseBadRequest()
	} else {
		// パースできたら、リクエストハンドラを呼びます。
		res, err = requesthandlers.OnRequest(req)
		if err != nil {
			// ハンドラ内でエラーが起きたら Internal Server Error を返します。
			log.Printf("[info] OnRequest() failed\n%s\n", err.Error())
			res = requesthandlers.ResponseInternalServerError()
		}
	}

	// レスポンスである shiori.Response をGo string にします。
	// 前述の通り、レスポンスの文字コードは UTF-8 に決め打ちです。
	var res_str string
	res.Headers["Charset"] = "UTF-8"
	res_str = res.String()

	log.Printf("[info] response content [%s]\n%s", res.Charset(), res)

	// Go string の res_str を C で扱えるように、char の配列にします。
	// C.CString() は malloc() してメモリを確保し、そこに Go string の内容をコピーする関数です。
	res_buf := C.CString(res_str)

	// C.CString() で確保したメモリは自前で free() してやる必要があります。
	// この時、res_buf の型は *C.char なので unsafe.Pointer (つまり void*) にキャストします。
	defer C.free((unsafe.Pointer)(res_buf))

	// バッファのサイズを調べます。
	// len(res_str) でもいいんですが、後でキャストの手間が少しだけ省けるので strlen() を呼んでいます。
	res_size := C.strlen(res_buf)

	// 調べたサイズを基に、レスポンス用のメモリを確保します。
	// SIZE_T は Win32 API での size_t です。
	ret := C.GlobalAlloc(C.GPTR, (C.SIZE_T)(res_size))

	// 確保したメモリにレスポンスをコピーします。
	// ret の型は C.HGLOBAL (つまり void*) なのですが、明示的に unsafe.Pointer にキャストしてやらねばなりません。
	// 逆に res_size は strlen() を使ったために型が C.size_t となり、キャストする必要がありません。
	C.memcpy((unsafe.Pointer)(ret), (unsafe.Pointer)(res_buf), res_size)

	// レスポンスのサイズを request() の第 2引数である length ポインタが指す先にキャストして書き込んでやります。
	*length = (C.long)(res_size)

	// レスポンスが入ったメモリ領域へのポインタ（HGLOBAL = void* です）を返して終了！
	return ret
}

package main

import (
    "io"
    "net/http"
    "log"
    "os"
    "strconv"
    "io/ioutil"
    "encoding/json"
    "errors"
    "strings"
)

const LISTEN_ADDRESS = ":9203"

var config Config

type Config []struct {
   ApiUrl string `json:"apiUrl"`
   AccountId string `json:"accountId"`
}

type FlyPoolMinerStatistics struct {
    Status string `json:"status"`
    Data struct {
        Time int `json:"time"`
        LastSeen int `json:"lastSeen"`
        ReportedHashrate float64 `json:"reportedHashrate"`
        CurrentHashrate float64 `json:"currentHashrate"`
        AverageHashrate float64 `json:"averageHashrate"`
        ActiveWorkers int `json:"activeWorkers"`
        Unpaid int `json:"unpaid"`
        Unconfirmed int `json:"unconfirmed"`
        ValidShares int `json:"validShares"`
        InvalidShares int `json:"invalidShares"`
        StaleShares int `json:"staleShares"`
        CoinsPerMin float64 `json:"coinsPerMin"`
        UsdPerMin float64 `json:"usdPerMin"`
        BtcPerMin float64 `json:"btcPerMin"`
    } `json:"data"`
}

func floatToString(value float64, precision int) string {
    return strconv.FormatFloat(value, 'f', precision, 64)
}

func integerToString(value int) string {
    return strconv.Itoa(value)
}

func convertBaseUnits(value string, precision int) string {
    if (len(value) < precision) {
        value = strings.Repeat("0", precision - len(value)) + value;
    }
    return value[:len(value)-precision+1] + "." + value[len(value)-precision+1:]
}

func formatValue(key string, meta string, value string) string {
    result := key;
    if (meta != "") {
        result += "{" + meta + "}";
    }
    result += " "
    result += value
    result += "\n"
    return result
}

func getConfig() (Config, error) {
    dir, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }
    body, err := ioutil.ReadFile(dir + "/config.json")
    if err != nil {
        log.Fatal(err)
    }

    bodyStr := string(body)

    jsonData := Config{}
    json.Unmarshal([]byte(bodyStr), &jsonData)

    return jsonData, nil
}

func queryData(apiUrl string, accountId string) (string, error) {
    // Build URL
    url := apiUrl + "/miner/" + accountId + "/currentStats"

    // Perform HTTP request
    resp, err := http.Get(url);
    if err != nil {
        return "", err;
    }

    // Parse response
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return "", errors.New("HTTP returned code " + integerToString(resp.StatusCode))
    }
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    bodyString := string(bodyBytes)
    if err != nil {
        return "", err;
    }

    return bodyString, nil;
}

func metrics(w http.ResponseWriter, r *http.Request) {
    log.Print("Serving /metrics")

    var up int
    var jsonString string
    var err error

    log.Print(config)

    for _, miner := range config {
        up = 1

        // Query miner statistics
        jsonString, err = queryData(miner.ApiUrl, miner.AccountId)
        if err != nil {
            log.Print(err)
            up = 0
        }

        // Parse JSON
        jsonData := FlyPoolMinerStatistics{}
        json.Unmarshal([]byte(jsonString), &jsonData)

        // Check response status
        if (jsonData.Status != "OK") {
            log.Print("Received negative status in JSON response '" + jsonData.Status + "'")
            log.Print(jsonString)
            up = 0
        }

        // Output
        io.WriteString(w, formatValue("flypool_up", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", integerToString(up)))
        io.WriteString(w, formatValue("flypool_time", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", integerToString(jsonData.Data.Time)))
        io.WriteString(w, formatValue("flypool_lastseen", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", integerToString(jsonData.Data.LastSeen)))
        io.WriteString(w, formatValue("flypool_hashrate_reported", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", floatToString(jsonData.Data.ReportedHashrate, 6)))
        io.WriteString(w, formatValue("flypool_hashrate_current", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", floatToString(jsonData.Data.CurrentHashrate, 6)))
        io.WriteString(w, formatValue("flypool_hashrate_average", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", floatToString(jsonData.Data.AverageHashrate, 6)))
        io.WriteString(w, formatValue("flypool_active_workers", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", integerToString(jsonData.Data.ActiveWorkers)))
        io.WriteString(w, formatValue("flypool_balance_unpaid", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", convertBaseUnits(integerToString(jsonData.Data.Unpaid), 19)))
        io.WriteString(w, formatValue("flypool_balance_unconfirmed", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", convertBaseUnits(integerToString(jsonData.Data.Unconfirmed), 19)))
        io.WriteString(w, formatValue("flypool_shares_valid", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", integerToString(jsonData.Data.ValidShares)))
        io.WriteString(w, formatValue("flypool_shares_invalid", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", integerToString(jsonData.Data.InvalidShares)))
        io.WriteString(w, formatValue("flypool_shares_stale", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", integerToString(jsonData.Data.StaleShares)))
        io.WriteString(w, formatValue("flypool_coins_per_min", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", floatToString(jsonData.Data.CoinsPerMin, 19)))
        io.WriteString(w, formatValue("flypool_usd_per_min", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", floatToString(jsonData.Data.UsdPerMin, 19)))
        io.WriteString(w, formatValue("flypool_btc_per_min", "apiUrl=\"" + miner.ApiUrl + "\",account=\"" + miner.AccountId + "\"", floatToString(jsonData.Data.BtcPerMin, 19)))
    }
}

func index(w http.ResponseWriter, r *http.Request) {
    log.Print("Serving /index")
    html := `<!doctype html>
<html>
    <head>
        <meta charset="utf-8">
        <title>Flypool Exporter</title>
    </head>
    <body>
        <h1>Flypool Exporter</h1>
        <p><a href="/metrics">Metrics</a></p>
    </body>
</html>`
    io.WriteString(w, html)
}

func main() {
    var err error
    config, err = getConfig()
    if err != nil {
        log.Fatal(err)
    }

    log.Print("Flypool exporter listening on " + LISTEN_ADDRESS)
    http.HandleFunc("/", index)
    http.HandleFunc("/metrics", metrics)
    http.ListenAndServe(LISTEN_ADDRESS, nil)
}

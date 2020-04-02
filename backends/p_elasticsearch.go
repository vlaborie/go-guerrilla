package backends

import (
    "fmt"
    "bytes"
    "context"
    "net/textproto"
    "encoding/json"

    "github.com/flashmob/go-guerrilla/mail"

    "github.com/elastic/go-elasticsearch/v7"
    "github.com/elastic/go-elasticsearch/v7/esapi"
)

func init() {
    processors["elasticsearch"] = func() Decorator {
        return Elasticsearch()
    }
}

type ElasticsearchAddress struct {
    User string `json:"user"`
    Domain string `json:"domain"`
}

type ElasticsearchEnvelope struct {
    // Remote IP address
    RemoteIP string `json:"remote_ip"`
    // Message sent in EHLO command
    Helo string `json:"helo"`
    // Sender
    MailFrom ElasticsearchAddress `json:"from"`
    // Recipients
    RcptTo ElasticsearchAddress `json:"recipient"`
    // Data stores the header and message body
    Data string `json:"data"`
    // Subject stores the subject of the email, extracted and decoded after calling ParseHeaders()
    Subject string `json:"subject"`
    // TLS is true if the email was received using a TLS connection
    TLS bool `json:"tls"`
    // Header stores the results from ParseHeaders()
    Header textproto.MIMEHeader `json:"header"`
    // additional delivery header that may be added
    DeliveryHeader string `json:"delivery_header"`
    // Email(s) will be queued with this id
    QueuedId string `json:"queue_id"`
    // ESMTP: true if EHLO was used
    ESMTP bool `json:"esmtp"`
}

type ElasticsearchProcessor struct {
    isConnected bool
    conn *elasticsearch.Client
}

func (e *ElasticsearchProcessor) elasticsearchConnection() (err error) {
    if e.isConnected == false {
        e.conn, err = elasticsearch.NewDefaultClient()
        if err != nil {
           return err
        }
        e.isConnected = true
    }
    return nil
}

func Elasticsearch() Decorator {
    elasticsearchClient := &ElasticsearchProcessor{}

    // Connect to Elasticsearch
    Svc.AddInitializer(InitializeWith(func(backendConfig BackendConfig) error {
        if elasticsearchErr := elasticsearchClient.elasticsearchConnection(); elasticsearchErr != nil {
            err := fmt.Errorf("elasticsearch cannot connect, check your settings: %s", elasticsearchErr)
            return err
        }
        return nil
    }))

    // When shutting down
///    Svc.AddShutdowner(ShutdownWith(func() error {
///        if elasticsearchClient.isConnected {
///            return nil
///        }
///        return nil
///    }))

    return func(p Processor) Processor {
        return ProcessWith(func(e *mail.Envelope, task SelectTask) (Result, error) {
            if task == TaskSaveMail {
                for i, _ := range e.RcptTo {
                    var from = ElasticsearchAddress {
                        User: e.MailFrom.User,
                        Domain: e.MailFrom.Host,
                    }
                    var recipient = ElasticsearchAddress {
                        User: e.RcptTo[i].User,
                        Domain: e.RcptTo[i].Host,
                    }

                    // Create ElasticsearchEnvelope from mail.Envelope
                    var ElasticsearchEnvelope = ElasticsearchEnvelope {
                        RemoteIP: e.RemoteIP,
                        Helo: e.Helo,
                        MailFrom: from,
                        RcptTo: recipient,
                        Data: e.Data.String(),
                        Subject: e.Subject,
                        TLS: e.TLS,
                        Header: e.Header,
                        DeliveryHeader: e.DeliveryHeader,
                        QueuedId: e.QueuedId,
                        ESMTP: e.ESMTP,
                    }

                    // prepare Elasticsearch request
                    req := esapi.IndexRequest{
                        Index: "test",
                        DocumentID: e.Hashes[i],
                        Refresh: "true",
                    }
                    j, _ := json.Marshal(ElasticsearchEnvelope)
                    req.Body = bytes.NewReader(j)

                    // perform the request
                    res, err := req.Do(context.Background(), elasticsearchClient.conn)
                    if err != nil {
                        Log().WithError(err).Warn("Error while uploading to elasticsearch")
                    }
                    defer res.Body.Close()
                }
                return p.Process(e, task)
            } else {
                return p.Process(e, task)
            }
        })
    }
}

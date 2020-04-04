// +build default full attachments_parser

package backends

import (
	"github.com/flashmob/go-guerrilla/mail"
)

// ----------------------------------------------------------------------------------
// Processor Name: attachmentsparser
// ----------------------------------------------------------------------------------
// Description   : Parses attachments using e.ParseAttachments()
// ----------------------------------------------------------------------------------
// Config Options: none
// --------------:-------------------------------------------------------------------
// Input         : envelope
// ----------------------------------------------------------------------------------
// Output        : Attachments will be populated in e.Attachments
// ----------------------------------------------------------------------------------
func init() {
	processors["attachmentsparser"] = func() Decorator {
		return AttachmentsParser()
	}
}

func AttachmentsParser() Decorator {
	return func(p Processor) Processor {
		return ProcessWith(func(e *mail.Envelope, task SelectTask) (Result, error) {
			if task == TaskSaveMail {
				if err := e.ParseAttachments(); err != nil {
					Log().WithError(err).Error("parse attachments error")
				}
				// next processor
				return p.Process(e, task)
			} else {
				// next processor
				return p.Process(e, task)
			}
		})
	}
}

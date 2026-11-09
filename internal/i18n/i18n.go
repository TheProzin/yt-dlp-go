package i18n

import "strings"

type Lang string

const (
	English      Lang = "en"
	PortugueseBR Lang = "pt-BR"
)

var current = English

// Set selects the UI language. Unknown values fall back to English.
func Set(l string) {
	switch strings.ToLower(strings.ReplaceAll(l, "_", "-")) {
	case "pt-br", "pt", "portuguese", "portugues":
		current = PortugueseBR
	default:
		current = English
	}
}

func Current() Lang { return current }

func IsPortuguese() bool { return current == PortugueseBR }

// T translates an English key to the current language.
// English is the default and the fallback when a key is missing.
func T(key string) string {
	if current == PortugueseBR {
		if v, ok := ptBR[key]; ok {
			return v
		}
	}
	return key
}

var ptBR = map[string]string{
	// Tabs / general
	"Download":            "Baixar",
	"History":             "Histórico",
	"Settings":            "Configurações",
	"Diagnostics":         "Diagnóstico",
	"Open":                "Abrir",
	"Choose...":           "Escolher...",
	"Cancel":              "Cancelar",
	"Refresh":             "Atualizar",
	"Repeat":              "Repetir",
	"Detect":              "Detectar",
	"Save settings":       "Salvar configurações",
	"Check now":           "Verificar agora",
	"Formats...":          "Formatos...",
	"Log":                 "Log",
	"Warning":             "Aviso",
	"OK":                  "OK",
	"FAIL":                "FALHA",
	"Audio":               "Áudio",
	"Use":                 "Usar",
	"Choose format":       "Escolher formato",
	"Automatic (preset)":  "Automático (preset)",
	"None":                "Nenhum",
	"English":             "Inglês",
	"Portuguese (Brazil)": "Português (Brasil)",
	"video":               "vídeo",

	// Window
	"yt-dlp-go - yt-dlp manager": "yt-dlp-go - Gerenciador yt-dlp",

	// Download tab
	"URLs (one per line)":                      "URLs (uma por linha)",
	"Paste one or more URLs (one per line)...": "Cole uma ou mais URLs (uma por linha)...",
	"Output folder":                            "Pasta de saída",
	"Quality:":                                 "Qualidade:",
	"Prefer H.264 (compatible)":                "Preferir H.264 (compatível)",
	"Cookies:":                                 "Cookies:",
	"Format: automatic (preset)":               "Formato: automático (preset)",
	"Format: %s":                               "Formato: %s",
	"Could not open %q: %v":                    "Não foi possível abrir %q: %v",
	"Best quality":                             "Melhor qualidade",
	"Audio only (MP3)":                         "Somente áudio (MP3)",
	"No download in progress.":                 "Nenhum download em andamento.",
	"Cancelling...":                            "Cancelando...",
	"Running yt-dlp...":                        "Executando yt-dlp...",
	"Starting %d download(s) (%d in parallel) in %s": "Iniciando %d download(s) (%d em paralelo) em %s",
	"Parallel downloads":                             "Downloads em paralelo",
	"Expanding playlist %s":                          "Expandindo playlist %s",
	"Found %d video(s)":                              "Encontrados %d vídeo(s)",
	"no videos found":                                "nenhum vídeo encontrado",
	"[%d/%d] failed: %v":                             "[%d/%d] falhou: %v",
	"[%d/%d] completed":                              "[%d/%d] concluído",
	"Finished.":                                      "Finalizado.",
	"Cancelled.":                                     "Cancelado.",
	"Finished with error: %s":                        "Finalizado com erro: %s",
	"END.":                                           "FIM.",
	"CANCELLED.":                                     "CANCELADO.",
	"ERROR: %s":                                      "ERRO: %s",
	"Downloads finished.":                            "Downloads concluídos.",
	"Download failed: %s":                            "Falha no download: %s",
	"Enter at least one URL.":                        "Informe ao menos uma URL.",
	"Choose the output folder.":                      "Escolha a pasta de saída.",
	"Fetching formats...":                            "Buscando formatos...",
	"Failed to fetch formats.":                       "Falha ao buscar formatos.",
	"Formats loaded.":                                "Formatos carregados.",
	"Enter a URL to list formats.":                   "Informe uma URL para listar os formatos.",

	// Advanced
	"Subtitles":                         "Legendas",
	"Download subtitles":                "Baixar legendas",
	"Include auto-generated subtitles":  "Incluir legendas automáticas",
	"Languages":                         "Idiomas",
	"Embed subtitles in video":          "Embutir legendas no vídeo",
	"Thumbnail":                         "Miniatura",
	"Download thumbnail":                "Baixar miniatura",
	"Embed thumbnail":                   "Embutir miniatura",
	"Metadata":                          "Metadados",
	"Embed metadata":                    "Embutir metadados",
	"Embed chapters":                    "Embutir capítulos",
	"Save metadata file (.info.json)":   "Salvar arquivo de metadados (.info.json)",
	"Name and organization":             "Nome e organização",
	"Template":                          "Template",
	"Organize into channel subfolder":   "Organizar em subpasta por canal",
	"Windows-compatible names":          "Nomes compatíveis com Windows",
	"Playlist":                          "Playlist",
	"Single video":                      "Vídeo único",
	"Full playlist":                     "Playlist completa",
	"Range":                             "Intervalo",
	"Network":                           "Rede",
	"Limit":                             "Limite",
	"Proxy":                             "Proxy",
	"Retries":                           "Tentativas",
	"Concurrent fragments":              "Fragmentos simultâneos",
	"Duplicates":                        "Duplicados",
	"Record downloaded (avoid repeats)": "Registrar baixados (evitar repetir)",
	"Do not overwrite existing files":   "Não sobrescrever arquivos existentes",
	"Example: 1-5,8,10-":                "ex: 1-5,8,10-",
	"Example: 5M (empty = no limit)":    "ex: 5M (vazio = sem limite)",
	"Example: socks5://127.0.0.1:9050":  "ex: socks5://127.0.0.1:9050",

	// History
	"History of downloads":         "Histórico de downloads",
	"History unavailable: %s":      "Histórico indisponível: %s",
	"Open folder":                  "Abrir pasta",
	"No items in history.":         "Nenhum item no histórico.",
	"%s  [%s]  %s  (%d items)\n%s": "%s  [%s]  %s  (%d itens)\n%s",
	"running":                      "em andamento",
	"ok":                           "ok",
	"error":                        "erro",
	"canceled":                     "cancelado",
	"interrupted":                  "interrompido",

	// Settings
	"Videos folder":                          "Pasta de vídeos",
	"Subtitles subfolder":                    "Subpasta de legendas",
	"Thumbnails subfolder":                   "Subpasta de miniaturas",
	"Metadata subfolder":                     "Subpasta de metadados",
	"Filename template":                      "Template de nome",
	"Default preset":                         "Preset padrão",
	"Cookies browser":                        "Navegador de cookies",
	"yt-dlp path":                            "Caminho do yt-dlp",
	"Language":                               "Idioma",
	"empty = auto-detect":                    "vazio = detectar automaticamente",
	"Update yt-dlp":                          "Atualizar yt-dlp",
	"Binary not found.":                      "Binário não encontrado.",
	"Settings saved.":                        "Configurações salvas.",
	"Restart the app to apply the language.": "Reinicie o aplicativo para aplicar o idioma.",

	// Diagnostics
	"Checking...":       "Verificando...",
	"Free space":        "Espaço livre",
	"Folder permission": "Permissão da pasta",
	"not found":         "não encontrado",
	"not found (needed to merge/extract audio)":      "não encontrado (necessário para mesclar/extrair áudio)",
	"not found (enables high resolutions in yt-dlp)": "não encontrado (libera resoluções altas no yt-dlp)",
	"not found (needed for login cookies)":           "não encontrado (necessário para cookies de login)",
	"writable":                                       "gravável",
	"installed":                                      "instalado",

	// openPath errors
	"empty path":                "caminho vazio",
	"folder does not exist: %w": "pasta não existe: %w",
	"no folder opener found":    "nenhum abridor de pastas encontrado",

	// downloader / history errors
	"yt-dlp not found: %w":                          "yt-dlp não encontrado: %w",
	"no URLs provided":                              "nenhuma URL informada",
	"output folder not defined":                     "pasta de saída não definida",
	"could not create folder %s: %w":                "não foi possível criar a pasta %s: %w",
	"a download is already running":                 "já existe um download em andamento",
	"invalid json: %w":                              "json inválido: %w",
	"another yt-dlp-go instance is already running": "outra instância do yt-dlp-go já está aberta",
	"%d of %d downloads failed: %w":                 "%d de %d downloads falharam: %w",
}

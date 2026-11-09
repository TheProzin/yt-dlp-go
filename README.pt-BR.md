# yt-dlp-go

Gerenciador desktop (Fyne) para o `yt-dlp`.

**Idiomas:** [English](README.md) · Português (Brasil) (este arquivo)

## Aviso legal e uso pessoal

Esta ferramenta foi criada para **uso pessoal e privado** e deve ser usada
apenas para baixar conteudos:

- de **dominio publico**;
- cuja **licenca permita** o download e/ou a redistribuicao;
- cujo download seja **expressamente autorizado** pelo autor ou pela plataforma;
- ou para os quais **voce detenha os direitos**.

O autor do yt-dlp-go **nao incentiva nem apoia** a violacao de direitos autorais
ou dos termos de servico de terceiros. Voce e o **unico responsavel** pelo uso
da ferramenta e pelo conteudo que baixar. O software e fornecido **"como esta"**,
sem garantias de qualquer natureza, e o autor nao se responsabiliza por qualquer
uso indevido, perda de dados ou consequencia legal decorrente do uso.

> Resumo: e uma ferramenta pessoal para conteudo livre/autorizado. Respeite os
> direitos autorais e os termos de cada site.

## O que faz

- Baixa uma ou varias URLs (uma por linha).
- Escolhe a pasta de saida e onde salvar os arquivos auxiliares.
- Qualidade: presets (Melhor, 1080p, 720p, 480p, somente audio MP3) e opcao de
  escolher formatos reais via `yt-dlp -J`.
- Preferencia por **H.264** (compativel com players antigos como VLC 3.x).
- Cookies do navegador (Chrome, Firefox, etc.) para login / anti-bot.
- Opcoes avancadas: legendas, miniatura, metadados/capitulos, nome e organizacao,
  playlists, rede (limite, proxy, tentativas) e arquivo para evitar duplicados.
- Historico persistente, diagnostico de dependencias e log em tempo real.
- **Downloads em paralelo**: concorrencia configuravel (padrao 5) para varios
  links e itens de playlist.
- Cancelamento mata a arvore de processos (inclusive `ffmpeg`).
- **Idioma da interface: ingles por padrao, pt-BR opcional** (Configuracoes → Idioma).

## Requisitos

- Go 1.25+ (apenas para compilar do fonte; o pacote pronto ja inclui o binario)
- `yt-dlp` no PATH ou em `~/.local/bin/yt-dlp` (ou `~/.config/yt-dlg/yt-dlp.real`)
- `deno` em `~/.deno/bin` (yt-dlp usa para resolver desafios JS / liberar 1080p)
- `ffmpeg` para mesclar audio/video e converter para MP3
- Navegador logado no YouTube (se usar cookies)

## Instalacao

### Ubuntu / Debian (.deb)

Baixe o `yt-dlp-go_1.1.0_amd64.deb` e instale com duplo clique ou:

```bash
sudo apt install ./yt-dlp-go_1.1.0_amd64.deb
```

Tambem funciona com `sudo dpkg -i yt-dlp-go_1.1.0_amd64.deb`.
Depois e so abrir **yt-dlp-go** no menu.

Para remover:

```bash
sudo apt remove yt-dlp-go
```

### Qualquer distro Linux (AppImage)

O AppImage e um arquivo unico, sem instalacao:

```bash
chmod +x yt-dlp-go-1.1.0-amd64.AppImage
./yt-dlp-go-1.1.0-amd64.AppImage
```

> Em distros sem FUSE 2, rode com `--appimage-extract-and-run`.
> No Ubuntu 24.04: `sudo apt install libfuse2t64` (ou use a opcao acima).

### Desenvolvimento

```bash
go run .                 # roda direto
go build -o yt-dlp-go .  # gera o binario
```

### Gerar os pacotes (.deb + AppImage)

```bash
./package.sh             # requer dpkg-deb e baixa o appimagetool na 1a vez
# saida em dist/:
#   yt-dlp-go_1.1.0_amd64.deb
#   yt-dlp-go-1.1.0-amd64.AppImage
```

## Onde salvar os arquivos

Tudo parte da **pasta de videos** escolhida na aba Baixar. Por padrao, os
arquivos auxiliares vao para subpastas automaticas:

```
<Videos>/<nome do video>
<Videos>/legendas/...
<Videos>/miniaturas/...
<Videos>/metadados/...
```

A pasta `metadados/` e usada quando **Salvar arquivo de metadados (.info.json)** estiver marcado.
Os nomes das subpastas podem ser alterados em **Configuracoes**.

## Persistencia

- `~/.config/yt-dlp-go/settings.json` - preferencias (incluindo idioma).
- `~/.config/yt-dlp-go/history.json` - historico (permissao 0600).
- `~/.config/yt-dlp-go/.lock` - trava de instancia unica.
- `~/.config/yt-dlp-go/download-archive.txt` - arquivo de downloads (se ativado).

## Observacoes

- O binario do yt-dlp e localizado nesta ordem: `$YTDLP_BIN`, `~/.local/bin/yt-dlp`,
  `~/.config/yt-dlg/yt-dlp.real`, `~/.config/yt-dlg/yt-dlp`, depois `PATH`.
- A aplicacao passa `--ignore-config`, entao ela nao herda o `~/.config/yt-dlp/config`.
- O app exige login no navegador para videos com checagem anti-bot do YouTube.
- Playlists completas sao expandidas em videos individuais para baixar em
  paralelo; campos de template especificos de playlist podem ficar indisponiveis.

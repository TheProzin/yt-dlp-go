#!/usr/bin/env bash
# Gera pacotes distribuiveis para Linux: .deb e .AppImage
set -euo pipefail

APP_NAME="yt-dlp-go"
BIN_NAME="yt-dlp-go"
APP_ID="br.com.yt-dlp-go"
VERSION="${VERSION:-1.1.0}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST="$ROOT/dist"

ARCH_GO="$(go env GOARCH)"
ARCH_DEB="$(dpkg --print-architecture)"

cd "$ROOT"
mkdir -p "$DIST"

echo "Compilando $APP_NAME $VERSION ($ARCH_GO)..."
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o "$DIST/$BIN_NAME" .


compute_deps() {
	local pkgs
	pkgs="$(ldd "$DIST/$BIN_NAME" 2>/dev/null \
		| awk '{for(i=1;i<=NF;i++) if($i ~ /^\//) print $i}' \
		| while IFS= read -r lib; do
			real="$(readlink -f "$lib" 2>/dev/null || echo "$lib")"
			dpkg -S "$real" 2>/dev/null | head -1 | cut -d: -f1
		done | sort -u | paste -sd, -)"
	if [ -z "$pkgs" ]; then
		pkgs="libc6, libgl1, libx11-6, libxcursor1, libxrandr2, libxinerama1, libxi6, libxxf86vm1"
	fi
	echo "$pkgs"
}

build_deb() {
	local root="$DIST/.deb-root"
	rm -rf "$root"
	install -d -m 755 "$root/DEBIAN" \
		"$root/usr" \
		"$root/usr/bin" \
		"$root/usr/share" \
		"$root/usr/share/applications" \
		"$root/usr/share/icons" \
		"$root/usr/share/icons/hicolor" \
		"$root/usr/share/icons/hicolor/512x512" \
		"$root/usr/share/icons/hicolor/512x512/apps"

	install -m 755 "$DIST/$BIN_NAME" "$root/usr/bin/$BIN_NAME"
	install -m 644 "$ROOT/assets/icon.png" "$root/usr/share/icons/hicolor/512x512/apps/$APP_ID.png"

	cat > "$root/usr/share/applications/$APP_ID.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=$APP_NAME
Comment=yt-dlp download manager
Comment[pt_BR]=Gerenciador de downloads do yt-dlp
Exec=/usr/bin/$BIN_NAME
Icon=$APP_ID
Terminal=false
Categories=AudioVideo;
Keywords=yt-dlp;youtube;download;video;
StartupWMClass=$BIN_NAME
EOF

	local size
	size="$(du -sk "$root/usr" | cut -f1)"
	local deps
	deps="$(compute_deps)"

	cat > "$root/DEBIAN/control" <<EOF
Package: $BIN_NAME
Version: $VERSION
Section: video
Priority: optional
Architecture: $ARCH_DEB
Installed-Size: $size
Maintainer: yt-dlp-go <noreply@localhost>
Depends: $deps
Recommends: yt-dlp, ffmpeg
Description: Gerenciador desktop para o yt-dlp
 Interface grafica (Fyne) para baixar videos com o yt-dlp,
 com historico, escolha de formatos e configuracoes.
EOF

	cat > "$root/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
if command -v update-desktop-database >/dev/null 2>&1; then
	update-desktop-database -q /usr/share/applications || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
	gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi
exit 0
EOF

	cat > "$root/DEBIAN/postrm" <<'EOF'
#!/bin/sh
set -e
if command -v update-desktop-database >/dev/null 2>&1; then
	update-desktop-database -q /usr/share/applications || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
	gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi
exit 0
EOF

	chmod 644 "$root/usr/share/applications/$APP_ID.desktop" "$root/DEBIAN/control"
	chmod 755 "$root/DEBIAN/postinst" "$root/DEBIAN/postrm"
	(cd "$root" && find usr -type f -print0 | sort -z | xargs -0 md5sum > DEBIAN/md5sums)
	chmod 644 "$root/DEBIAN/md5sums"

	dpkg-deb --root-owner-group --build "$root" "$DIST/${BIN_NAME}_${VERSION}_${ARCH_DEB}.deb" >/dev/null
	echo "DEB:     $DIST/${BIN_NAME}_${VERSION}_${ARCH_DEB}.deb"
}

build_appimage() {
	local appdir="$DIST/$APP_NAME.AppDir"
	rm -rf "$appdir"
	mkdir -p "$appdir/usr/bin" \
		"$appdir/usr/share/applications" \
		"$appdir/usr/share/icons/hicolor/512x512/apps"

	install -m 755 "$DIST/$BIN_NAME" "$appdir/usr/bin/$BIN_NAME"
	install -m 644 "$ROOT/assets/icon.png" "$appdir/usr/share/icons/hicolor/512x512/apps/$APP_ID.png"
	install -m 644 "$ROOT/assets/icon.png" "$appdir/$BIN_NAME.png"
	ln -sf "$BIN_NAME.png" "$appdir/.DirIcon"

	cat > "$appdir/$BIN_NAME.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=$APP_NAME
Comment=yt-dlp download manager
Comment[pt_BR]=Gerenciador de downloads do yt-dlp
Exec=$BIN_NAME
Icon=$BIN_NAME
Terminal=false
Categories=AudioVideo;
Keywords=yt-dlp;youtube;download;video;
StartupWMClass=$BIN_NAME
EOF
	cp "$appdir/$BIN_NAME.desktop" "$appdir/usr/share/applications/$APP_ID.desktop"

	cat > "$appdir/AppRun" <<EOF
#!/bin/sh
HERE="\$(dirname "\$(readlink -f "\$0")")"
exec "\$HERE/usr/bin/$BIN_NAME" "\$@"
EOF
	chmod +x "$appdir/AppRun"

	local tool="$HOME/.cache/yt-dlp-go/appimagetool-x86_64.AppImage"
	if [ ! -x "$tool" ]; then
		mkdir -p "$(dirname "$tool")"
		echo "Baixando appimagetool..."
		curl -fL -o "$tool" \
			"https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
		chmod +x "$tool"
	fi

	APPIMAGE_EXTRACT_AND_RUN=1 ARCH=x86_64 "$tool" "$appdir" "$DIST/${BIN_NAME}-${VERSION}-${ARCH_GO}.AppImage" >/dev/null
	echo "AppImage: $DIST/${BIN_NAME}-${VERSION}-${ARCH_GO}.AppImage"
}

build_deb
build_appimage

rm -rf "$DIST/.deb-root" "$DIST/$APP_NAME.AppDir" "$DIST/$BIN_NAME"

echo "Concluido. Pacotes em: $DIST"

package main

// HTML templates for web interface using TailAdmin-inspired layout
// Uses Go template composition to avoid code duplication

// Shared head template with Tailwind config
const headHTML = `
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <script defer src="https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js"></script>
    <script src="/static/config-modal.js"></script>
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: {
                        brand: {
                            50: '#eff6ff',
                            100: '#dbeafe',
                            200: '#bfdbfe',
                            300: '#93c5fd',
                            400: '#60a5fa',
                            500: '#3b82f6',
                            600: '#2563eb',
                            700: '#1d4ed8',
                            800: '#1e40af',
                            900: '#1e3a8a',
                            950: '#172554',
                        },
                    }
                }
            }
        }
    </script>
    <style>
        [x-cloak] { display: none !important; }
    </style>
    <script>if (localStorage.getItem('darkMode') === 'true') { document.documentElement.classList.add('dark'); }</script>
`

// Config modal JavaScript - served at /static/config-modal.js
const configModalJS = `
let serverIPValue = '';

function showConfigModal() {
    document.getElementById('configModal').classList.remove('hidden');
    document.getElementById('configModal').classList.add('flex');
    fetch('/api/server-info')
        .then(r => r.json())
        .then(data => {
            serverIPValue = data.ip || window.location.hostname;
            document.getElementById('configModalServerIP').textContent = serverIPValue;
            document.querySelectorAll('.config-ip').forEach(el => {
                el.textContent = serverIPValue;
            });
        })
        .catch(() => {
            serverIPValue = window.location.hostname;
            document.getElementById('configModalServerIP').textContent = serverIPValue;
            document.querySelectorAll('.config-ip').forEach(el => {
                el.textContent = serverIPValue;
            });
        });
}

function hideConfigModal() {
    document.getElementById('configModal').classList.add('hidden');
    document.getElementById('configModal').classList.remove('flex');
}

function copyServerIP() {
    const ip = serverIPValue;
    
    // Check if navigator and clipboard are available
    if (typeof navigator !== 'undefined' && navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(ip).then(() => {
            showCopyFeedback();
        }).catch(err => {
            console.warn('Clipboard API failed, trying fallback:', err);
            fallbackCopy(ip);
        });
    } else {
        // Fallback for older browsers or non-HTTPS
        fallbackCopy(ip);
    }
}

function fallbackCopy(text) {
    // Create a temporary textarea element
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    document.body.appendChild(textArea);
    
    try {
        textArea.focus();
        textArea.select();
        const successful = document.execCommand('copy');
        if (successful) {
            showCopyFeedback();
        } else {
            alert('Failed to copy IP address. Please copy manually: ' + text);
        }
    } catch (err) {
        console.error('Fallback copy failed:', err);
        alert('Failed to copy IP address. Please copy manually: ' + text);
    } finally {
        document.body.removeChild(textArea);
    }
}

function showCopyFeedback() {
    const btn = event.currentTarget;
    const originalHTML = btn.innerHTML;
    btn.innerHTML = '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>';
    setTimeout(() => { btn.innerHTML = originalHTML; }, 1500);
}
`

// Config modal HTML - to be included in all pages
const configModalHTML = `
    <!-- Configuration Modal -->
    <div id="configModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white dark:bg-gray-900 rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-xl font-bold">DNS Server Configuration</h2>
                <button onclick="hideConfigModal()" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                    </svg>
                </button>
            </div>
            <div class="space-y-4">
                <div class="p-4 bg-brand-50 dark:bg-brand-950 rounded-xl border border-brand-200 dark:border-brand-700">
                    <p class="text-sm text-brand-700 dark:text-brand-200 mb-2">Configure your devices to use this DNS server:</p>
                    <div class="flex items-center gap-3">
                        <code id="configModalServerIP" class="flex-1 px-4 py-3 bg-white dark:bg-gray-800 rounded-lg font-mono text-lg font-bold text-center">Loading...</code>
                        <button onclick="copyServerIP()" class="p-3 bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors" title="Copy to clipboard">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                            </svg>
                        </button>
                    </div>
                </div>
                
                <div class="space-y-3">
                    <h3 class="font-semibold text-sm text-gray-600 dark:text-gray-400 uppercase tracking-wide">Configuration Instructions</h3>
                    
                    <details class="group border border-gray-200 dark:border-gray-800 rounded-xl">
                        <summary class="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 rounded-xl">
                            <span class="font-medium">🐧 Linux</span>
                            <svg class="w-5 h-5 transition-transform group-open:rotate-180" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                            </svg>
                        </summary>
                        <div class="px-4 pb-4 text-sm text-gray-600 dark:text-gray-400 space-y-2">
                            <p>Edit <code class="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded">/etc/resolv.conf</code>:</p>
                            <pre class="p-3 bg-gray-100 dark:bg-gray-800 rounded-lg overflow-x-auto"><code>nameserver <span class="config-ip">SERVER_IP</span></code></pre>
                            <p>Or use NetworkManager/systemd-resolved for persistent configuration.</p>
                        </div>
                    </details>
                    
                    <details class="group border border-gray-200 dark:border-gray-800 rounded-xl">
                        <summary class="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 rounded-xl">
                            <span class="font-medium">🍎 macOS</span>
                            <svg class="w-5 h-5 transition-transform group-open:rotate-180" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                            </svg>
                        </summary>
                        <div class="px-4 pb-4 text-sm text-gray-600 dark:text-gray-400 space-y-2">
                            <p>System Preferences → Network → Advanced → DNS</p>
                            <p>Add <code class="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded config-ip">SERVER_IP</code> as DNS server.</p>
                        </div>
                    </details>
                    
                    <details class="group border border-gray-200 dark:border-gray-800 rounded-xl">
                        <summary class="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 rounded-xl">
                            <span class="font-medium">🪟 Windows</span>
                            <svg class="w-5 h-5 transition-transform group-open:rotate-180" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                            </svg>
                        </summary>
                        <div class="px-4 pb-4 text-sm text-gray-600 dark:text-gray-400 space-y-2">
                            <p>Control Panel → Network → Change adapter settings</p>
                            <p>Right-click adapter → Properties → IPv4 → Use the following DNS server:</p>
                            <p>Enter <code class="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded config-ip">SERVER_IP</code></p>
                        </div>
                    </details>
                    
                    <details class="group border border-gray-200 dark:border-gray-800 rounded-xl">
                        <summary class="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 rounded-xl">
                            <span class="font-medium">🌐 Router (Recommended)</span>
                            <svg class="w-5 h-5 transition-transform group-open:rotate-180" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                            </svg>
                        </summary>
                        <div class="px-4 pb-4 text-sm text-gray-600 dark:text-gray-400 space-y-2">
                            <p>Access your router's admin panel and set the DNS server to:</p>
                            <p><code class="px-1 py-0.5 bg-gray-100 dark:bg-gray-800 rounded config-ip">SERVER_IP</code></p>
                            <p class="text-green-600 dark:text-green-400">✓ This will apply to all devices on your network.</p>
                        </div>
                    </details>
                </div>
            </div>
            <div class="mt-6 flex justify-end">
                <button onclick="hideConfigModal()" class="px-4 py-2 bg-brand-600 text-white rounded-lg hover:bg-brand-700">Got it</button>
            </div>
        </div>
    </div>
`

// Header template - PageTitle determines the page title, ShowSetupButton shows setup button
const headerHTML = `{{define "header"}}
            <!-- Header -->
            <header class="sticky top-0 z-30 flex w-full bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
                <div class="flex flex-grow items-center justify-between px-4 py-4 md:px-6">
                    <div class="flex items-center gap-4">
                        <button @click="sidebarOpen = !sidebarOpen" class="lg:hidden p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">
                            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
                            </svg>
                        </button>
                        {{block "headerContent" .}}<h1 class="text-xl font-semibold">{{.PageTitle}}</h1>{{end}}
                    </div>
                    <div class="flex items-center gap-3">
                        {{if .ShowSetupButton}}
                        <button onclick="showConfigModal()" class="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-brand-600 dark:text-brand-400 border border-brand-600 dark:border-brand-400 rounded-lg hover:bg-brand-50 dark:hover:bg-brand-900/20 transition-colors">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                            </svg>
                            Setup me!
                        </button>
                        {{end}}
                        <button @click="darkMode = !darkMode" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">
                            <svg x-show="!darkMode" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
                            </svg>
                            <svg x-show="darkMode" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" x-cloak>
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/>
                            </svg>
                        </button>
                        <a href="/logout" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-white" title="Logout">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                            </svg>
                        </a>
                    </div>
                </div>
            </header>
{{end}}`

// Sidebar template - CurrentPath determines active link
const sidebarHTML = `{{define "sidebar"}}
        <aside :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full'"
               class="fixed left-0 top-0 z-50 flex h-screen w-72 flex-col overflow-y-hidden bg-gray-900 border-r border-gray-800 dark:bg-gray-900 dark:border-gray-800 duration-300 ease-linear lg:static lg:translate-x-0">
            
            <div class="flex items-center justify-between gap-2 px-6 py-5 lg:py-6">
                <a href="/" class="flex items-center gap-2">
                    <span class="text-2xl">🌐</span>
                    <span class="text-xl font-bold text-white">SimpleDNS</span>
                </a>
                <button @click="sidebarOpen = false" class="block lg:hidden text-gray-400">
                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                    </svg>
                </button>
            </div>

            <div class="flex flex-col overflow-y-auto duration-300 ease-linear no-scrollbar">
                <nav class="px-4 py-4">
                    <div>
                        <h3 class="mb-4 text-xs font-semibold uppercase tracking-wider text-gray-400">Menu</h3>
                        <ul class="space-y-2">
                            <li>
                                <a href="/" class="flex items-center gap-3 px-4 py-3 rounded-lg {{if eq .CurrentPath "/"}}bg-brand-600 text-white{{else}}text-gray-300 hover:bg-white/5 hover:text-white{{end}}">
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                                        <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 0 1 6 3.75h2.25A2.25 2.25 0 0 1 10.5 6v2.25a2.25 2.25 0 0 1-2.25 2.25H6a2.25 2.25 0 0 1-2.25-2.25V6ZM3.75 15.75A2.25 2.25 0 0 1 6 13.5h2.25a2.25 2.25 0 0 1 2.25 2.25V18a2.25 2.25 0 0 1-2.25 2.25H6A2.25 2.25 0 0 1 3.75 18v-2.25ZM13.5 6a2.25 2.25 0 0 1 2.25-2.25H18A2.25 2.25 0 0 1 20.25 6v2.25A2.25 2.25 0 0 1 18 10.5h-2.25a2.25 2.25 0 0 1-2.25-2.25V6ZM13.5 15.75a2.25 2.25 0 0 1 2.25-2.25H18a2.25 2.25 0 0 1 2.25 2.25V18A2.25 2.25 0 0 1 18 20.25h-2.25A2.25 2.25 0 0 1 13.5 18v-2.25Z" />
                                    </svg>
                                    <span>Overview</span>
                                </a>
                            </li>
                            <li>
                                <a href="/zones" class="flex items-center gap-3 px-4 py-3 rounded-lg {{if eq .CurrentPath "/zones"}}bg-brand-600 text-white{{else}}text-gray-300 hover:bg-white/5 hover:text-white{{end}}">
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
                                    </svg>
                                    <span>Zones</span>
                                </a>
                            </li>
                            <li>
                                <a href="/forwarders" class="flex items-center gap-3 px-4 py-3 rounded-lg {{if eq .CurrentPath "/forwarders"}}bg-brand-600 text-white{{else}}text-gray-300 hover:bg-white/5 hover:text-white{{end}}">
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                                        <path stroke-linecap="round" stroke-linejoin="round" d="m5.25 4.5 7.5 7.5-7.5 7.5m6-15 7.5 7.5-7.5 7.5" />
                                    </svg>
                                    <span>Forwarders</span>
                                </a>
                            </li>
                            <li>
                                <a href="/replication" class="flex items-center gap-3 px-4 py-3 rounded-lg {{if eq .CurrentPath "/replication"}}bg-brand-600 text-white{{else}}text-gray-300 hover:bg-white/5 hover:text-white{{end}}">
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                                        <path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21 3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" />
                                    </svg>
                                    <span>Replication</span>
                                </a>
                            </li>
                        </ul>
                    </div>
                    <div class="mt-6">
                        <h3 class="mb-4 text-xs font-semibold uppercase tracking-wider text-gray-400">Account</h3>
                        <ul class="space-y-2">
                            <li>
                                <a href="/account" class="flex items-center gap-3 px-4 py-3 rounded-lg {{if eq .CurrentPath "/account"}}bg-brand-600 text-white{{else}}text-gray-300 hover:bg-white/5 hover:text-white{{end}}">
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                                        <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z" />
                                    </svg>
                                    <span>Profile</span>
                                </a>
                            </li>
                            <li>
                                <a href="/account/tokens" class="flex items-center gap-3 px-4 py-3 rounded-lg {{if eq .CurrentPath "/account/tokens"}}bg-brand-600 text-white{{else}}text-gray-300 hover:bg-white/5 hover:text-white{{end}}">
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                                        <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 0 1 3 3m3 0a6 6 0 0 1-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1 1 21.75 8.25Z" />
                                    </svg>
                                    <span>API Tokens</span>
                                </a>
                            </li>
                        </ul>
                    </div>
                </nav>
            </div>

            <div class="mt-auto px-4 py-4 border-t border-gray-200 dark:border-gray-800">
                <div class="flex items-center justify-between text-sm text-gray-400">
                    <span>Version {{.Version}} • Mode: {{.Mode}}</span>
                    <a href="/api/health" class="hover:text-gray-900 dark:hover:text-white">API</a>
                </div>
            </div>
        </aside>
{{end}}`

// Index page - Dashboard with zones list
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - Zones</title>
` + headHTML + `
</head>
<body x-data="{ sidebarOpen: false, darkMode: localStorage.getItem('darkMode') === 'true' }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans">
    
    <div class="flex h-screen overflow-hidden">
        {{template "sidebar" .}}

        <!-- Content Area -->
        <div class="relative flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
            
            <div x-show="sidebarOpen" @click="sidebarOpen = false" 
                 class="fixed inset-0 z-40 bg-black/50 lg:hidden" x-cloak></div>

            {{template "header" .}}

            <!-- Main Content -->
            <main class="p-4 md:p-6 2xl:p-10">
                <!-- Zones Table -->
                <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03] overflow-hidden">
                    <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800 flex justify-between items-center">
                        <h3 class="text-lg font-semibold">DNS Zones</h3>
                        {{if .EditMode}}
                        <button onclick="showAddZoneModal()" class="flex items-center gap-2 px-4 py-2 text-sm bg-brand-600 text-white hover:bg-brand-700 rounded-lg transition-colors">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                            </svg>
                            Add Domain
                        </button>
                        {{end}}
                    </div>
                    {{if .Zones}}
                    <div class="overflow-x-auto">
                        <table class="w-full">
                            <thead class="border-b border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-white/[0.02]">
                                <tr>
                                    <th class="px-5 py-3 sm:px-6 text-left">
                                        <span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Zone Name</span>
                                    </th>
                                    <th class="px-5 py-3 sm:px-6 text-left">
                                        <span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Status</span>
                                    </th>
                                    <th class="px-5 py-3 sm:px-6 text-left">
                                        <span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Records</span>
                                    </th>
                                    <th class="px-5 py-3 sm:px-6 text-right">
                                        <span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Actions</span>
                                    </th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                                {{range .Zones}}
                                <tr>
                                    <td class="px-5 py-4 sm:px-6">
                                        <a href="/zones/{{.Name}}/records" class="font-medium text-gray-800 text-sm dark:text-white/90 hover:text-brand-600 dark:hover:text-brand-400 hover:underline">{{.Name}}</a>
                                    </td>
                                    <td class="px-5 py-4 sm:px-6">
                                        {{if .Enabled}}
                                        <div class="flex items-center gap-2">
                                            <span class="flex h-3 w-3 rounded-full bg-green-500"></span>
                                            <span class="text-sm text-green-600 dark:text-green-400">Active</span>
                                        </div>
                                        {{else}}
                                        <div class="flex items-center gap-2">
                                            <span class="flex h-3 w-3 rounded-full bg-red-500"></span>
                                            <span class="text-sm text-red-600 dark:text-red-400">Disabled</span>
                                        </div>
                                        {{end}}
                                    </td>
                                    <td class="px-5 py-4 sm:px-6">
                                        <span class="text-sm text-gray-600 dark:text-gray-300">{{len .Records}}</span>
                                    </td>
                                    <td class="px-5 py-4 sm:px-6">
                                        <div class="flex items-center justify-end gap-2">
                                            <a href="/zones/{{.Name}}/records" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5" title="View Records">
                                                <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                                                </svg>
                                            </a>
                                            <a href="/zones/{{.Name}}/settings" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5" title="Settings">
                                                <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                                                </svg>
                                            </a>
                                        </div>
                                    </td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                    {{else}}
                    <div class="p-10 text-center text-gray-500 dark:text-gray-400">
                        <svg class="mx-auto w-12 h-12 mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/>
                        </svg>
                        <p class="text-lg font-medium">No zones configured</p>
                        {{if .EditMode}}<p class="text-sm mt-2">Click "Add Zone" to create your first zone.</p>{{end}}
                    </div>
                    {{end}}
                </div>

                </main>
        </div>
    </div>

    {{if .EditMode}}
    <!-- Add Zone Modal -->
    <div id="addZoneModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white dark:bg-gray-900 rounded-2xl p-6 w-full max-w-md mx-4 shadow-xl">
            <h2 class="text-xl font-bold mb-4">Add New Zone</h2>
            <form id="addZoneForm" onsubmit="submitZone(event)">
                <div class="mb-4">
                    <label class="block text-sm font-medium mb-2">Zone Name</label>
                    <input type="text" name="name" required placeholder="example.com" 
                           class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                </div>
                <div class="flex gap-3 justify-end">
                    <button type="button" onclick="hideAddZoneModal()" class="px-4 py-2 border border-gray-300 dark:border-gray-800 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">Cancel</button>
                    <button type="submit" class="px-4 py-2 bg-brand-600 text-white rounded-lg hover:bg-brand-700">Create Zone</button>
                </div>
            </form>
        </div>
    </div>
    {{end}}

    <script>
        function showAddZoneModal() {
            document.getElementById('addZoneModal').classList.remove('hidden');
            document.getElementById('addZoneModal').classList.add('flex');
        }
        function hideAddZoneModal() {
            document.getElementById('addZoneModal').classList.add('hidden');
            document.getElementById('addZoneModal').classList.remove('flex');
            document.getElementById('addZoneForm').reset();
        }
        
        async function submitZone(event) {
            event.preventDefault();
            const form = event.target;
            try {
                const resp = await fetch('/api/zones', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({ name: form.name.value })
                });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    const err = await resp.json();
                    alert('Failed to create zone: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
` + configModalHTML + `
</body>
</html>
`

// Replication token support removed; no client-side token JS

// Zone Records page
const zoneRecordsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - {{.Zone.Name}} Records</title>
` + headHTML + `
</head>
<body x-data="{ sidebarOpen: false, darkMode: localStorage.getItem('darkMode') === 'true', activeFilter: 'all', searchQuery: '' }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans">
    
    <div class="flex h-screen overflow-hidden">
        {{template "sidebar" .}}

        <!-- Content Area -->
        <div class="relative flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
            
            <div x-show="sidebarOpen" @click="sidebarOpen = false" 
                 class="fixed inset-0 z-40 bg-black/50 lg:hidden" x-cloak></div>

            <!-- Header -->
            <header class="sticky top-0 z-30 flex w-full bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
                <div class="flex flex-grow items-center justify-between px-4 py-4 md:px-6">
                    <div class="flex items-center gap-4">
                        <button @click="sidebarOpen = !sidebarOpen" class="lg:hidden p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">
                            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
                            </svg>
                        </button>
                        <nav class="flex items-center gap-2 text-sm">
                            <select onchange="if(this.value) window.location.href='/zones/' + this.value + '/records'" 
                                    class="font-medium bg-transparent border border-gray-300 dark:border-gray-700 rounded-lg px-3 py-1.5 pr-8 focus:outline-none focus:ring-2 focus:ring-brand-500 cursor-pointer appearance-none"
                                    style="background-image: url('data:image/svg+xml;charset=UTF-8,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 fill=%22none%22 viewBox=%220 0 24 24%22 stroke=%22%236b7280%22%3E%3Cpath stroke-linecap=%22round%22 stroke-linejoin=%22round%22 stroke-width=%222%22 d=%22M19 9l-7 7-7-7%22/%3E%3C/svg%3E'); background-repeat: no-repeat; background-position: right 0.5rem center; background-size: 1rem;">
                                {{range .AllZones}}
                                <option value="{{.Name}}" {{if eq .Name $.Zone.Name}}selected{{end}}>{{.Name}}</option>
                                {{end}}
                            </select>
                        </nav>
                    </div>
                    <div class="flex items-center gap-3">
                        <button @click="darkMode = !darkMode" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">
                            <svg x-show="!darkMode" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
                            </svg>
                            <svg x-show="darkMode" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" x-cloak>
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/>
                            </svg>
                        </button>
                        <a href="/logout" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-white" title="Logout">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                            </svg>
                        </a>
                    </div>
                </div>
            </header>

            <!-- Main Content -->
            <main class="p-4 md:p-6 2xl:p-10">
                <!-- Zone Header -->
                <div class="mb-6">
                    <div class="flex items-center gap-3 mb-2">
                        <h1 class="text-2xl font-bold">{{.Zone.Name}}</h1>
                        {{if .Zone.Enabled}}
                        <span class="px-2.5 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 rounded-full">Active</span>
                        {{else}}
                        <span class="px-2.5 py-0.5 text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 rounded-full">Disabled</span>
                        {{end}}
                    </div>
                    <p class="text-gray-500 dark:text-gray-400 mb-4">{{len .Zone.Records}} DNS records</p>
                    
                    <!-- Tabs with underline and icon -->
                    <div class="border-b border-gray-200 dark:border-gray-800">
                        <nav class="flex gap-6">
                            <a href="/zones/{{.Zone.Name}}/records" class="flex items-center gap-2 px-1 pb-3 border-b-2 border-brand-600 text-brand-600 dark:text-brand-400 font-medium text-sm">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                                </svg>
                                Records
                            </a>
                            <a href="/zones/{{.Zone.Name}}/settings" class="flex items-center gap-2 px-1 pb-3 border-b-2 border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300 font-medium text-sm">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                                </svg>
                                Settings
                            </a>
                        </nav>
                    </div>
                </div>

                <!-- Filter Buttons -->
                <div class="flex flex-wrap items-center gap-4 mb-4">
                    <div class="flex flex-wrap gap-2">
                        <template x-for="filter in ['all', 'A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'PTR']">
                            <button @click="activeFilter = filter"
                                    :class="activeFilter === filter ? 'bg-brand-600 text-white' : 'bg-white dark:bg-white/[0.03] border border-gray-300 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-white/5'"
                                    class="px-3 py-1.5 text-sm rounded-lg transition-colors"
                                    x-text="filter === 'all' ? 'All' : filter">
                            </button>
                        </template>
                    </div>
                    <div class="relative flex-1 min-w-[200px] max-w-md">
                        <input type="text" x-model="searchQuery" placeholder="Search records..."
                               class="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500 text-sm">
                        <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                        </svg>
                    </div>
                </div>

                <!-- Records Table -->
                <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03] overflow-hidden">
                    <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800 flex justify-between items-center">
                        <h3 class="text-lg font-semibold">DNS Records</h3>
                        {{if .EditMode}}
                        <button onclick="showAddRecordModal()" class="flex items-center gap-2 px-4 py-2 text-sm bg-brand-600 text-white hover:bg-brand-700 rounded-lg transition-colors">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                            </svg>
                            Add Record
                        </button>
                        {{end}}
                    </div>
                    {{if .Zone.Records}}
                    <div class="overflow-x-auto">
                        <table class="w-full">
                            <thead class="border-b border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-white/[0.02]">
                                <tr>
                                    <th class="px-5 py-3 sm:px-6 text-left"><span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Name</span></th>
                                    <th class="px-5 py-3 sm:px-6 text-left"><span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Type</span></th>
                                    <th class="px-5 py-3 sm:px-6 text-left"><span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Value</span></th>
                                    <th class="px-5 py-3 sm:px-6 text-left"><span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Priority</span></th>
                                    <th class="px-5 py-3 sm:px-6 text-left"><span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">TTL</span></th>
                                    {{if .EditMode}}<th class="px-5 py-3 sm:px-6 text-right"><span class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Actions</span></th>{{end}}
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                                {{range .Zone.Records}}
                                <tr x-show="(activeFilter === 'all' || activeFilter === '{{.Type}}') && (searchQuery === '' || '{{.Name}} {{.Value}}'.toLowerCase().includes(searchQuery.toLowerCase()))">
                                    <td class="px-5 py-4 sm:px-6"><span class="font-mono text-sm" data-field="name">{{.Name}}</span></td>
                                    <td class="px-5 py-4 sm:px-6">
                                        <span class="px-2 py-1 text-xs font-medium rounded
                                            {{if eq .Type "A"}}bg-blue-100 text-blue-800 dark:bg-blue-500/20 dark:text-blue-300
                                            {{else if eq .Type "AAAA"}}bg-indigo-100 text-indigo-800 dark:bg-indigo-500/20 dark:text-indigo-300
                                            {{else if eq .Type "CNAME"}}bg-green-100 text-green-800 dark:bg-green-500/20 dark:text-green-300
                                            {{else if eq .Type "MX"}}bg-purple-100 text-purple-800 dark:bg-purple-500/20 dark:text-purple-300
                                            {{else if eq .Type "TXT"}}bg-yellow-100 text-yellow-800 dark:bg-yellow-500/20 dark:text-yellow-300
                                            {{else if eq .Type "NS"}}bg-pink-100 text-pink-800 dark:bg-pink-500/20 dark:text-pink-300
                                            {{else if eq .Type "PTR"}}bg-orange-100 text-orange-800 dark:bg-orange-500/20 dark:text-orange-300
                                            {{else}}bg-gray-100 text-gray-800 dark:bg-gray-500/20 dark:text-gray-300{{end}}" data-field="type">{{.Type}}</span>
                                    </td>
                                    <td class="px-5 py-4 sm:px-6"><span class="font-mono text-sm text-gray-600 dark:text-gray-300 break-all" data-field="value">{{.Value}}</span></td>
                                    <td class="px-5 py-4 sm:px-6"><span class="text-sm text-gray-500" data-field="priority">{{if eq .Type "MX"}}{{.Priority}}{{else}}-{{end}}</span></td>
                                    <td class="px-5 py-4 sm:px-6"><span class="text-sm text-gray-500" data-field="ttl">{{.TTL}}</span></td>
                                    {{if $.EditMode}}
                                    <td class="px-5 py-4 sm:px-6">
                                        <div class="flex items-center justify-end gap-2">
                                            <button onclick="showEditRecordModal({{.ID}}, this)" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5" title="Edit">
                                                <svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                                                </svg>
                                            </button>
                                            <button onclick="deleteRecord({{.ID}}, this)" class="p-2 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20" title="Delete">
                                                <svg class="w-4 h-4 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                                </svg>
                                            </button>
                                        </div>
                                    </td>
                                    {{end}}
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                    {{else}}
                    <div class="p-10 text-center text-gray-500 dark:text-gray-400">
                        <p class="text-lg font-medium">No records in this zone</p>
                        {{if .EditMode}}<p class="text-sm mt-2">Click "Add Record" to create your first record.</p>{{end}}
                    </div>
                    {{end}}
                </div>
            </main>
        </div>
    </div>

    {{if .EditMode}}
    <!-- Add Record Modal -->
    <div id="addRecordModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white dark:bg-gray-900 rounded-2xl p-6 w-full max-w-md mx-4 shadow-xl">
            <h2 class="text-xl font-bold mb-4">Add DNS Record</h2>
            <form id="addRecordForm" onsubmit="submitRecord(event)">
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium mb-2">Name</label>
                        <input type="text" name="name" required placeholder="subdomain or @" 
                               class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-2">Type</label>
                        <select name="type" required class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                            <option value="A">A</option>
                            <option value="AAAA">AAAA</option>
                            <option value="CNAME">CNAME</option>
                            <option value="MX">MX</option>
                            <option value="TXT">TXT</option>
                            <option value="NS">NS</option>
                            <option value="PTR">PTR</option>
                        </select>
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-2">Value</label>
                        <input type="text" name="value" required placeholder="192.168.1.1" 
                               class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    </div>
                    <div id="priorityFieldAdd" style="display: none;">
                        <label class="block text-sm font-medium mb-2">Priority (MX only)</label>
                        <input type="number" name="priority" value="10" min="0" max="65535"
                               class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-2">TTL</label>
                        <input type="number" name="ttl" value="3600" min="60" 
                               class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    </div>
                </div>
                <div class="flex gap-3 justify-end mt-6">
                    <button type="button" onclick="hideAddRecordModal()" class="px-4 py-2 border border-gray-300 dark:border-gray-800 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">Cancel</button>
                    <button type="submit" class="px-4 py-2 bg-brand-600 text-white rounded-lg hover:bg-brand-700">Add Record</button>
                </div>
            </form>
        </div>
    </div>

    <!-- Edit Record Modal -->
    <div id="editRecordModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white dark:bg-gray-900 rounded-2xl p-6 w-full max-w-md mx-4 shadow-xl">
            <h2 class="text-xl font-bold mb-4">Edit DNS Record</h2>
            <form id="editRecordForm" onsubmit="submitEditRecord(event)">
                <input type="hidden" id="editRecordId">
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium mb-2">Name</label>
                        <input type="text" id="editRecordName" required 
                               class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-2">Type</label>
                        <select id="editRecordType" required class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                            <option value="A">A</option>
                            <option value="AAAA">AAAA</option>
                            <option value="CNAME">CNAME</option>
                            <option value="MX">MX</option>
                            <option value="TXT">TXT</option>
                            <option value="NS">NS</option>
                            <option value="PTR">PTR</option>
                        </select>
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-2">Value</label>
                        <input type="text" id="editRecordValue" required 
                               class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    </div>
                    <div id="priorityFieldEdit" style="display: none;">
                        <label class="block text-sm font-medium mb-2">Priority (MX only)</label>
                        <input type="number" id="editRecordPriority" value="10" min="0" max="65535"
                               class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    </div>
                    <div>
                        <label class="block text-sm font-medium mb-2">TTL</label>
                        <input type="number" id="editRecordTTL" min="60" 
                               class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-800 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    </div>
                </div>
                <div class="flex gap-3 justify-end mt-6">
                    <button type="button" onclick="hideEditRecordModal()" class="px-4 py-2 border border-gray-300 dark:border-gray-800 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">Cancel</button>
                    <button type="submit" class="px-4 py-2 bg-brand-600 text-white rounded-lg hover:bg-brand-700">Save Changes</button>
                </div>
            </form>
        </div>
    </div>
    {{end}}

    <script>
        const zoneId = {{.Zone.ID}};
        
        // Toggle priority field visibility based on record type
        function togglePriorityField(selectElement, fieldId) {
            const priorityField = document.getElementById(fieldId);
            if (selectElement.value === 'MX') {
                priorityField.style.display = 'block';
            } else {
                priorityField.style.display = 'none';
            }
        }
        
        // Add event listeners for type selects
        document.addEventListener('DOMContentLoaded', function() {
            const addTypeSelect = document.querySelector('#addRecordForm select[name="type"]');
            if (addTypeSelect) {
                addTypeSelect.addEventListener('change', function() {
                    togglePriorityField(this, 'priorityFieldAdd');
                });
            }
            const editTypeSelect = document.getElementById('editRecordType');
            if (editTypeSelect) {
                editTypeSelect.addEventListener('change', function() {
                    togglePriorityField(this, 'priorityFieldEdit');
                });
            }
        });
        
        function showAddRecordModal() {
            document.getElementById('addRecordModal').classList.remove('hidden');
            document.getElementById('addRecordModal').classList.add('flex');
            document.getElementById('priorityFieldAdd').style.display = 'none';
        }
        function hideAddRecordModal() {
            document.getElementById('addRecordModal').classList.add('hidden');
            document.getElementById('addRecordModal').classList.remove('flex');
            document.getElementById('addRecordForm').reset();
            document.getElementById('priorityFieldAdd').style.display = 'none';
        }
        
        async function submitRecord(event) {
            event.preventDefault();
            const form = event.target;
            const data = {
                zone_id: zoneId,
                name: form.name.value,
                type: form.type.value,
                value: form.value.value,
                ttl: parseInt(form.ttl.value) || 3600,
                priority: form.type.value === 'MX' ? (parseInt(form.priority.value) || 10) : 0
            };
            try {
                const resp = await fetch('/api/zones/' + zoneId + '/records', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    const err = await resp.json();
                    alert('Failed to add record: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        function showEditRecordModal(id, btn) {
            const row = btn.closest('tr');
            document.getElementById('editRecordId').value = id;
            document.getElementById('editRecordName').value = row.querySelector('[data-field="name"]').textContent.trim();
            const recordType = row.querySelector('[data-field="type"]').textContent.trim();
            document.getElementById('editRecordType').value = recordType;
            document.getElementById('editRecordValue').value = row.querySelector('[data-field="value"]').textContent.trim();
            document.getElementById('editRecordTTL').value = row.querySelector('[data-field="ttl"]').textContent.trim();
            const priorityText = row.querySelector('[data-field="priority"]').textContent.trim();
            document.getElementById('editRecordPriority').value = priorityText === '-' ? 10 : parseInt(priorityText) || 10;
            document.getElementById('priorityFieldEdit').style.display = recordType === 'MX' ? 'block' : 'none';
            document.getElementById('editRecordModal').classList.remove('hidden');
            document.getElementById('editRecordModal').classList.add('flex');
        }
        
        function hideEditRecordModal() {
            document.getElementById('editRecordModal').classList.add('hidden');
            document.getElementById('editRecordModal').classList.remove('flex');
            document.getElementById('priorityFieldEdit').style.display = 'none';
        }
        
        async function submitEditRecord(event) {
            event.preventDefault();
            const id = document.getElementById('editRecordId').value;
            const recordType = document.getElementById('editRecordType').value;
            const data = {
                name: document.getElementById('editRecordName').value,
                type: recordType,
                value: document.getElementById('editRecordValue').value,
                ttl: parseInt(document.getElementById('editRecordTTL').value) || 3600,
                priority: recordType === 'MX' ? (parseInt(document.getElementById('editRecordPriority').value) || 10) : 0
            };
            try {
                const resp = await fetch('/api/records/' + id, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    const err = await resp.json();
                    alert('Failed to update record: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        async function deleteRecord(id, btn) {
            if (!confirm('Delete this record?')) return;
            try {
                const resp = await fetch('/api/records/' + id, { method: 'DELETE' });
                if (resp.ok) {
                    btn.closest('tr').remove();
                } else {
                    alert('Failed to delete record');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
` + configModalHTML + `
</body>
</html>
`

// Zone Settings page
const zoneSettingsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - {{.Zone.Name}} Settings</title>
` + headHTML + `
</head>
<body x-data="{ sidebarOpen: false, darkMode: localStorage.getItem('darkMode') === 'true' }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans">
    
    <div class="flex h-screen overflow-hidden">
        {{template "sidebar" .}}

        <!-- Content Area -->
        <div class="relative flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
            
            <div x-show="sidebarOpen" @click="sidebarOpen = false" 
                 class="fixed inset-0 z-40 bg-black/50 lg:hidden" x-cloak></div>

            <!-- Header -->
            <header class="sticky top-0 z-30 flex w-full bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
                <div class="flex flex-grow items-center justify-between px-4 py-4 md:px-6">
                    <div class="flex items-center gap-4">
                        <button @click="sidebarOpen = !sidebarOpen" class="lg:hidden p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">
                            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
                            </svg>
                        </button>
                        <nav class="flex items-center gap-2 text-sm">
                            <select onchange="if(this.value) window.location.href='/zones/' + this.value + '/settings'" 
                                    class="font-medium bg-transparent border border-gray-300 dark:border-gray-700 rounded-lg px-3 py-1.5 pr-8 focus:outline-none focus:ring-2 focus:ring-brand-500 cursor-pointer appearance-none"
                                    style="background-image: url('data:image/svg+xml;charset=UTF-8,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 fill=%22none%22 viewBox=%220 0 24 24%22 stroke=%22%236b7280%22%3E%3Cpath stroke-linecap=%22round%22 stroke-linejoin=%22round%22 stroke-width=%222%22 d=%22M19 9l-7 7-7-7%22/%3E%3C/svg%3E'); background-repeat: no-repeat; background-position: right 0.5rem center; background-size: 1rem;">
                                {{range .AllZones}}
                                <option value="{{.Name}}" {{if eq .Name $.Zone.Name}}selected{{end}}>{{.Name}}</option>
                                {{end}}
                            </select>
                        </nav>
                    </div>
                    <div class="flex items-center gap-3">
                        <button @click="darkMode = !darkMode" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">
                            <svg x-show="!darkMode" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
                            </svg>
                            <svg x-show="darkMode" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" x-cloak>
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/>
                            </svg>
                        </button>
                        <a href="/logout" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-white" title="Logout">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                            </svg>
                        </a>
                    </div>
                </div>
            </header>

            <!-- Main Content -->
            <main class="p-4 md:p-6 2xl:p-10">
                <div class="mb-6">
                    <div class="flex items-center gap-3 mb-2">
                        <h1 class="text-2xl font-bold">{{.Zone.Name}}</h1>
                        {{if .Zone.Enabled}}
                        <span class="px-2.5 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 rounded-full">Active</span>
                        {{else}}
                        <span class="px-2.5 py-0.5 text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 rounded-full">Disabled</span>
                        {{end}}
                    </div>
                    <p class="text-gray-500 dark:text-gray-400 mb-4">{{len .Zone.Records}} DNS records</p>
                    
                    <!-- Tabs with underline and icon -->
                    <div class="border-b border-gray-200 dark:border-gray-800">
                        <nav class="flex gap-6">
                            <a href="/zones/{{.Zone.Name}}/records" class="flex items-center gap-2 px-1 pb-3 border-b-2 border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300 font-medium text-sm">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                                </svg>
                                Records
                            </a>
                            <a href="/zones/{{.Zone.Name}}/settings" class="flex items-center gap-2 px-1 pb-3 border-b-2 border-brand-600 text-brand-600 dark:text-brand-400 font-medium text-sm">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                                </svg>
                                Settings
                            </a>
                        </nav>
                    </div>
                </div>

                <!-- Zone Info -->
                <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03] mb-6">
                    <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800">
                        <h3 class="text-lg font-semibold">Zone Information</h3>
                    </div>
                    <div class="p-5">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div>
                                <label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Zone Name</label>
                                <p class="text-lg font-mono">{{.Zone.Name}}</p>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Records Count</label>
                                <p class="text-lg">{{len .Zone.Records}}</p>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Zone ID</label>
                                <p class="text-lg font-mono">{{.Zone.ID}}</p>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Status</label>
                                <div class="flex items-center gap-2">
                                    {{if .Zone.Enabled}}
                                    <span class="w-2 h-2 rounded-full bg-green-500"></span>
                                    <span class="text-green-600 dark:text-green-400">Active</span>
                                    {{else}}
                                    <span class="w-2 h-2 rounded-full bg-red-500"></span>
                                    <span class="text-red-600 dark:text-red-400">Disabled</span>
                                    {{end}}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                {{if .EditMode}}
                <!-- Danger Zone -->
                <div class="rounded-2xl border border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-900/10">
                    <div class="px-5 py-4 border-b border-red-200 dark:border-red-900/50">
                        <h3 class="text-lg font-semibold text-red-700 dark:text-red-400">Danger Zone</h3>
                    </div>
                    <div class="p-5">
                        <div class="flex items-center justify-between">
                            <div>
                                <h4 class="font-medium text-red-700 dark:text-red-400">Delete this zone</h4>
                                <p class="text-sm text-red-600/80 dark:text-red-400/80">Once you delete a zone, there is no going back. All records will be permanently deleted.</p>
                            </div>
                            <button onclick="deleteZone()" class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors">
                                Delete Zone
                            </button>
                        </div>
                    </div>
                </div>
                {{end}}
            </main>
        </div>
    </div>

    <script>
        const zoneId = {{.Zone.ID}};
        const zoneName = '{{.Zone.Name}}';
        
        async function deleteZone() {
            if (!confirm('Are you sure you want to delete zone ' + zoneName + '? This will remove all records and cannot be undone.')) return;
            if (!confirm('This is your last chance. Are you really sure?')) return;
            
            try {
                const resp = await fetch('/api/zones/' + zoneId, { method: 'DELETE' });
                if (resp.ok) {
                    window.location.href = '/';
                } else {
                    const err = await resp.json();
                    alert('Failed to delete zone: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
` + configModalHTML + `
</body>
</html>
`

// Global Settings page
const globalSettingsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - Overview</title>
` + headHTML + `
</head>
<body x-data="{ sidebarOpen: false, darkMode: localStorage.getItem('darkMode') === 'true' }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans">
    
    <div class="flex h-screen overflow-hidden">
        {{template "sidebar" .}}

        <!-- Content Area -->
        <div class="relative flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
            
            <div x-show="sidebarOpen" @click="sidebarOpen = false" 
                 class="fixed inset-0 z-40 bg-black/50 lg:hidden" x-cloak></div>

            {{template "header" .}}

            <!-- Main Content -->
            <main class="p-4 md:p-6 2xl:p-10 space-y-6">
                <!-- Stats Cards -->
                <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-2">
                    <article class="flex items-center gap-5 rounded-2xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
                        <div class="inline-flex h-16 w-16 items-center justify-center rounded-xl bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-white/90">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
                            </svg>
                        </div>
                        <div>
                            <h3 class="text-2xl font-semibold text-gray-800 dark:text-white/90">
                                {{.ZoneCount}}
                            </h3>
                            <p class="flex items-center gap-3 text-gray-500 dark:text-gray-400">
                                Total Zones
                            </p>
                        </div>
                    </article>
                    <article class="flex items-center gap-5 rounded-2xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
                        <div class="inline-flex h-16 w-16 items-center justify-center rounded-xl bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-white/90">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
                            </svg>
                        </div>
                        <div>
                            <h3 class="text-2xl font-semibold text-gray-800 dark:text-white/90">
                                {{.RecordCount}}
                            </h3>
                            <p class="flex items-center gap-3 text-gray-500 dark:text-gray-400">
                                Total Records
                            </p>
                        </div>
                    </article>
                </div>

                <!-- Server Info Section -->
                <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03]">
                    <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800">
                        <h3 class="text-lg font-semibold">Server informations</h3>
                    </div>
                    <div class="p-5">
                        <div class="grid grid-cols-1 md:grid-cols-4 gap-6">
                            <div>
                                <label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Server Role</label>
                                <div class="flex items-center gap-2">
                                    {{if eq .ServerRole "master"}}
                                    <span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                                        <svg class="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                                        </svg>
                                        Master
                                    </span>
                                    {{else}}
                                    <span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200">
                                        {{.ServerRole}}
                                    </span>
                                    {{end}}
                                </div>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Server IP Address</label>
                                <div class="flex items-center gap-2">
                                    <p class="text-lg font-mono" id="serverIP">Loading...</p>
                                    <button onclick="copyServerIP()" class="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-white/5" title="Copy IP">
                                        <svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                                        </svg>
                                    </button>
                                </div>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">DNS Port</label>
                                <p class="text-lg font-mono">{{.DNSPort}}</p>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Database Mode</label>
                                <p class="text-lg font-mono">{{.Mode}}</p>
                            </div>
                        </div>
                    </div>
                </div>

                <script>
                    // Fetch and display server IP
                    fetch('/api/server-info')
                        .then(r => r.json())
                        .then(data => {
                            document.getElementById('serverIP').textContent = data.ip || 'Unknown';
                        })
                        .catch(() => {
                            document.getElementById('serverIP').textContent = 'Error loading';
                        });
                    
                    function copyServerIP() {
                        const ip = document.getElementById('serverIP').textContent;
                        
                        // Try modern clipboard API first
                        if (navigator.clipboard && navigator.clipboard.writeText) {
                            navigator.clipboard.writeText(ip).then(() => {
                                alert('IP copied: ' + ip);
                            }).catch(err => {
                                console.warn('Clipboard API failed, trying fallback:', err);
                                fallbackCopy(ip);
                            });
                        } else {
                            // Fallback for older browsers or non-HTTPS
                            fallbackCopy(ip);
                        }
                    }
                    
                    function fallbackCopy(text) {
                        // Create a temporary textarea element
                        const textArea = document.createElement('textarea');
                        textArea.value = text;
                        textArea.style.position = 'fixed';
                        textArea.style.left = '-999999px';
                        textArea.style.top = '-999999px';
                        document.body.appendChild(textArea);
                        
                        try {
                            textArea.focus();
                            textArea.select();
                            const successful = document.execCommand('copy');
                            if (successful) {
                                alert('IP copied: ' + text);
                            } else {
                                alert('Failed to copy IP address. Please copy manually: ' + text);
                            }
                        } catch (err) {
                            console.error('Fallback copy failed:', err);
                            alert('Failed to copy IP address. Please copy manually: ' + text);
                        } finally {
                            document.body.removeChild(textArea);
                        }
                    }
                </script>
            </main>
        </div>
    </div>

` + configModalHTML + `
</body>
</html>
`

// Forwarders page template
const forwardersHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - Forwarders</title>
` + headHTML + `
</head>
<body x-data="{ sidebarOpen: false, darkMode: localStorage.getItem('darkMode') === 'true' }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans">
    
    <div class="flex h-screen overflow-hidden">
        {{template "sidebar" .}}

        <!-- Content Area -->
        <div class="relative flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
            
            <div x-show="sidebarOpen" @click="sidebarOpen = false" 
                 class="fixed inset-0 z-40 bg-black/50 lg:hidden" x-cloak></div>

            {{template "header" .}}

            <!-- Main Content -->
            <main class="p-4 md:p-6 2xl:p-10">
                <!-- Forwarders Section -->
                <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03]">
                    <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800 flex justify-between items-center">
                        <div>
                            <h3 class="text-lg font-semibold">DNS Forwarders</h3>
                            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Configure upstream DNS servers for queries that don't match any local zone</p>
                        </div>
                        {{if and .EditMode (lt (len .Forwarders) 2)}}
                        <button onclick="showAddForwarderModal()" class="flex items-center gap-2 px-4 py-2 text-sm bg-brand-600 text-white hover:bg-brand-700 rounded-lg transition-colors">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                            </svg>
                            Add Forwarder
                        </button>
                        {{else if and .EditMode (ge (len .Forwarders) 2)}}
                        <div class="flex items-center gap-2 px-4 py-2 text-sm text-gray-500 dark:text-gray-400">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"/>
                            </svg>
                            Maximum 2 forwarders allowed
                        </div>
                        {{end}}
                    </div>
                    <div class="p-5">
                        {{if .Forwarders}}
                        <div class="space-y-3" id="forwarders-list">
                            {{range .ForwarderDisplays}}
                            <div class="flex items-center justify-between px-4 py-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg" data-forwarder="{{.Address}}">
                                <div class="flex items-center gap-3">
                                    <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100 dark:bg-brand-900/20">
                                        <svg class="w-5 h-5 text-brand-600 dark:text-brand-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/>
                                        </svg>
                                    </div>
                                    <span class="font-mono text-sm">{{.Display}}</span>
                                </div>
                                {{if $.EditMode}}
                                <button onclick="deleteForwarder('{{.Address}}', this)" class="p-2 text-red-500 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors">
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                    </svg>
                                </button>
                                {{end}}
                            </div>
                            {{end}}
                        </div>
                        {{else}}
                        <div class="text-center py-10">
                            <svg class="mx-auto w-12 h-12 mb-4 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/>
                            </svg>
                            <p class="text-gray-500 dark:text-gray-400">No forwarders configured</p>
                            <p class="text-sm text-gray-400 dark:text-gray-500 mt-1">Add a forwarder to resolve external DNS queries</p>
                        </div>
                        {{end}}
                    </div>
                </div>
            </main>
        </div>
    </div>

    {{if .EditMode}}
    <!-- Add Forwarder Modal -->
    <div id="addForwarderModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white dark:bg-gray-900 rounded-2xl p-6 w-full max-w-md mx-4 shadow-xl">
            <h2 class="text-xl font-bold mb-4">Add Forwarder</h2>
            <form id="addForwarderForm" onsubmit="submitForwarder(event)">
                <div class="mb-4">
                    <label class="block text-sm font-medium mb-2">DNS Server Address</label>
                    <input type="text" name="address" required placeholder="8.8.8.8 or 8.8.8.8:53" 
                           class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-700 rounded-lg bg-white dark:bg-white/[0.03] focus:outline-none focus:ring-2 focus:ring-brand-500">
                    <p class="text-xs text-gray-500 mt-2">IP address or hostname, optionally with port (default: 53)</p>
                </div>
                <div class="flex gap-3 justify-end">
                    <button type="button" onclick="hideAddForwarderModal()" class="px-4 py-2 border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-white/5">Cancel</button>
                    <button type="submit" class="px-4 py-2 bg-brand-600 text-white rounded-lg hover:bg-brand-700">Add Forwarder</button>
                </div>
            </form>
        </div>
    </div>
    {{end}}

    <script>
        function showAddForwarderModal() {
            document.getElementById('addForwarderModal').classList.remove('hidden');
            document.getElementById('addForwarderModal').classList.add('flex');
        }
        
        function hideAddForwarderModal() {
            document.getElementById('addForwarderModal').classList.add('hidden');
            document.getElementById('addForwarderModal').classList.remove('flex');
            document.getElementById('addForwarderForm').reset();
        }
        
        async function submitForwarder(event) {
            event.preventDefault();
            const form = event.target;
            
            // Check if we already have 2 forwarders (client-side validation)
            const currentForwarders = document.querySelectorAll('[data-forwarder]');
            if (currentForwarders.length >= 2) {
                alert('Maximum 2 forwarders allowed');
                return;
            }
            
            let address = form.address.value.trim();
            if (!address.includes(':')) address = address + ':53';
            try {
                const resp = await fetch('/api/forwarders', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({ address: address })
                });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    const err = await resp.json();
                    alert('Failed to add forwarder: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        async function deleteForwarder(address, btn) {
            if (!confirm('Remove forwarder ' + address + '?')) return;
            try {
                const resp = await fetch('/api/forwarders/' + encodeURIComponent(address), { method: 'DELETE' });
                if (resp.ok) {
                    btn.closest('[data-forwarder]').remove();
                } else {
                    alert('Failed to remove forwarder');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
` + configModalHTML + `
</body>
</html>
`

// Replication page template
const replicationHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - Replication</title>
` + headHTML + `
</head>
<body x-data="{ sidebarOpen: false, darkMode: localStorage.getItem('darkMode') === 'true' }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans">
    <div class="flex h-screen overflow-hidden">
        {{template "sidebar" .}}

        <div class="relative flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
            <div x-show="sidebarOpen" @click="sidebarOpen = false" class="fixed inset-0 z-40 bg-black/50 lg:hidden" x-cloak></div>
            {{template "header" .}}

            <main class="p-4 md:p-6 2xl:p-10">
                <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03] p-6 mb-6">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-4">
                            <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-brand-100 dark:bg-brand-900/20">
                                <svg class="w-6 h-6 text-brand-600 dark:text-brand-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                                </svg>
                            </div>
                            <div>
                                <h3 class="text-lg font-semibold">Replication</h3>
                                <p class="text-sm text-gray-500 dark:text-gray-400">Replication status and controls</p>
                            </div>
                        </div>
                        <div>
                            {{if eq .ServerRole "master"}}
                            <span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">Master</span>
                            {{else}}
                            <span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200">{{.ServerRole}}</span>
                            {{end}}
                        </div>
                    </div>
                </div>
            </main>
        </div>
    </div>

` + configModalHTML + `
</body>
</html>
`

// Login page template
const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - Login</title>
` + headHTML + `
</head>
<body x-data="{ darkMode: localStorage.getItem('darkMode') === 'true' }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans min-h-screen flex items-center justify-center">
    
    <div class="w-full max-w-md px-6">
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-xl border border-gray-200 dark:border-gray-700 p-8">
            <!-- Logo -->
            <div class="text-center mb-8">
                <div class="flex items-center justify-center gap-3 mb-4">
                    <span class="text-4xl">🌐</span>
                    <span class="text-2xl font-bold">SimpleDNS</span>
                </div>
                <p class="text-gray-500 dark:text-gray-400">Sign in to your account</p>
            </div>

            {{if .Error}}
            <div class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                <p class="text-red-600 dark:text-red-400 text-sm text-center">{{.Error}}</p>
            </div>
            {{end}}

            <form method="POST" action="/login" class="space-y-6">
                <input type="hidden" name="redirect" value="{{.Redirect}}">
                
                <div>
                    <label for="username" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Username</label>
                    <input type="text" id="username" name="username" value="admin" readonly
                           class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-gray-100 dark:bg-gray-600 text-gray-500 dark:text-gray-400 cursor-not-allowed focus:ring-0 focus:border-gray-300 dark:focus:border-gray-600 outline-none transition"
                           required>
                </div>

                <div>
                    <label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Password</label>
                    <input type="password" id="password" name="password" 
                           class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-brand-500 focus:border-transparent outline-none transition"
                           required autofocus>
                </div>

                <button type="submit" 
                        class="w-full py-3 px-4 bg-brand-600 hover:bg-brand-700 text-white font-medium rounded-lg transition focus:ring-4 focus:ring-brand-300 dark:focus:ring-brand-800">
                    Sign In
                </button>
            </form>
        </div>

        <!-- Dark mode toggle -->
        <div class="mt-6 text-center">
            <button @click="darkMode = !darkMode" class="text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition">
                <span x-show="!darkMode" class="flex items-center gap-2">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
                    </svg>
                    <span class="text-sm">Dark Mode</span>
                </span>
                <span x-show="darkMode" x-cloak class="flex items-center gap-2">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/>
                    </svg>
                    <span class="text-sm">Light Mode</span>
                </span>
            </button>
            <div class="mt-4 text-xs text-gray-400 dark:text-gray-500">
                Version: {{.Version}}
            </div>
        </div>
    </div>
</body>
</html>
`

// Account/Password management page
const accountHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - Account</title>
` + headHTML + `
</head>
<body x-data="{ sidebarOpen: false, darkMode: localStorage.getItem('darkMode') === 'true', showPassword: false }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans">
    
    <div class="flex h-screen overflow-hidden">
        {{template "sidebar" .}}

        <!-- Content Area -->
        <div class="relative flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
            
            <div x-show="sidebarOpen" @click="sidebarOpen = false" 
                 class="fixed inset-0 z-40 bg-black/50 lg:hidden" x-cloak></div>

            {{template "header" .}}

            <!-- Main Content -->
            <main class="p-4 md:p-6 2xl:p-10">
                <div>
                    <!-- Success Message -->
                    {{if .Success}}
                    <div class="mb-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                        <p class="text-green-600 dark:text-green-400 text-sm">{{.Success}}</p>
                    </div>
                    {{end}}

                    <!-- Error Message -->
                    {{if .Error}}
                    <div class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                        <p class="text-red-600 dark:text-red-400 text-sm">{{.Error}}</p>
                    </div>
                    {{end}}

                    <!-- Account Info Card -->
                    <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03] mb-6">
                        <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800">
                            <h3 class="text-lg font-semibold">Account Information</h3>
                        </div>
                        <div class="p-5">
                            <div class="flex items-center gap-4">
                                <div class="flex h-16 w-16 items-center justify-center rounded-full bg-brand-100 dark:bg-brand-900/30">
                                    <svg class="w-8 h-8 text-brand-600 dark:text-brand-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                                    </svg>
                                </div>
                                <div>
                                    <p class="text-lg font-medium">{{.Username}}</p>
                                    <p class="text-sm text-gray-500 dark:text-gray-400">Administrator</p>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Change Password Card -->
                    <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03]">
                        <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800">
                            <h3 class="text-lg font-semibold">Change Password</h3>
                            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Update your account password</p>
                        </div>
                        <div class="p-5">
                            <form method="POST" action="/account" class="space-y-4">
                                <div>
                                    <label for="current_password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Current Password</label>
                                    <input type="password" id="current_password" name="current_password" 
                                           class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-brand-500 focus:border-transparent outline-none transition"
                                           required>
                                </div>

                                <div>
                                    <label for="new_password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">New Password</label>
                                    <div class="relative">
                                        <input :type="showPassword ? 'text' : 'password'" id="new_password" name="new_password" 
                                               class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-brand-500 focus:border-transparent outline-none transition pr-12"
                                               required minlength="8">
                                        <button type="button" @click="showPassword = !showPassword" 
                                                class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
                                            <svg x-show="!showPassword" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                                            </svg>
                                            <svg x-show="showPassword" x-cloak class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
                                            </svg>
                                        </button>
                                    </div>
                                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Minimum 8 characters</p>
                                </div>

                                <div>
                                    <label for="confirm_password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Confirm New Password</label>
                                    <input :type="showPassword ? 'text' : 'password'" id="confirm_password" name="confirm_password" 
                                           class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-brand-500 focus:border-transparent outline-none transition"
                                           required minlength="8">
                                </div>

                                <div class="pt-2">
                                    <button type="submit" 
                                            class="px-6 py-3 bg-brand-600 hover:bg-brand-700 text-white font-medium rounded-lg transition focus:ring-4 focus:ring-brand-300 dark:focus:ring-brand-800">
                                        Update Password
                                    </button>
                                </div>
                            </form>
                        </div>
                    </div>

                </div>
            </main>
        </div>
    </div>
` + configModalHTML + `
</body>
</html>
`

// API Tokens page template
const apiTokensHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - API Tokens</title>
` + headHTML + `
</head>
<body x-data="{ sidebarOpen: false, darkMode: localStorage.getItem('darkMode') === 'true' }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans">
    
    <div class="flex h-screen overflow-hidden">
        {{template "sidebar" .}}

        <!-- Content Area -->
        <div class="relative flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
            
            <div x-show="sidebarOpen" @click="sidebarOpen = false" 
                 class="fixed inset-0 z-40 bg-black/50 lg:hidden" x-cloak></div>

            {{template "header" .}}

            <!-- Main Content -->
            <main class="p-4 md:p-6 2xl:p-10">
                <!-- API Tokens Card -->
                <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03]">
                    <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800 flex justify-between items-center">
                        <div>
                            <h3 class="text-lg font-semibold">Manage API Tokens</h3>
                            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Create and manage tokens for programmatic API access</p>
                        </div>
                        <button onclick="showCreateTokenModal()" class="flex items-center gap-2 px-4 py-2 text-sm bg-brand-600 text-white hover:bg-brand-700 rounded-lg transition-colors">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                            </svg>
                            New Token
                        </button>
                    </div>
                    <div class="p-5">
                        {{if .APITokens}}
                        <div class="overflow-x-auto">
                            <table class="w-full">
                                <thead class="border-b border-gray-200 dark:border-gray-700">
                                    <tr>
                                        <th class="text-left py-3 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Name</th>
                                        <th class="text-left py-3 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Created</th>
                                        <th class="text-left py-3 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Last Used</th>
                                        <th class="text-right py-3 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Actions</th>
                                    </tr>
                                </thead>
                                <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                                    {{range .APITokens}}
                                    <tr>
                                        <td class="py-3">
                                            <div class="flex items-center gap-2">
                                                <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
                                                </svg>
                                                <span class="font-medium">{{.Name}}</span>
                                            </div>
                                        </td>
                                        <td class="py-3 text-sm text-gray-500 dark:text-gray-400">{{.CreatedAt}}</td>
                                        <td class="py-3 text-sm text-gray-500 dark:text-gray-400">{{if .LastUsedAt}}{{.LastUsedAt}}{{else}}Never{{end}}</td>
                                        <td class="py-3 text-right">
                                            <button onclick="deleteAPIToken({{.ID}})" class="p-2 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20" title="Delete">
                                                <svg class="w-4 h-4 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                                </svg>
                                            </button>
                                        </td>
                                    </tr>
                                    {{end}}
                                </tbody>
                            </table>
                        </div>
                        {{else}}
                        <div class="text-center py-8 text-gray-500 dark:text-gray-400">
                            <svg class="w-12 h-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
                            </svg>
                            <p>No API tokens yet</p>
                            <p class="text-sm mt-1">Create a token to access the API programmatically</p>
                        </div>
                        {{end}}
                    </div>
                </div>

                <!-- Usage Guide Card -->
                <div class="rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-white/[0.03] mt-6">
                    <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-800">
                        <h3 class="text-lg font-semibold">How to use API tokens</h3>
                    </div>
                    <div class="p-5 space-y-4">
                        <div>
                            <h4 class="text-sm font-medium mb-2">Authorization Header</h4>
                            <code class="block text-sm bg-gray-100 dark:bg-gray-900 p-3 rounded-lg font-mono">Authorization: Bearer sdns_your_token_here</code>
                        </div>
                        <div>
                            <h4 class="text-sm font-medium mb-2">X-API-Key Header</h4>
                            <code class="block text-sm bg-gray-100 dark:bg-gray-900 p-3 rounded-lg font-mono">X-API-Key: sdns_your_token_here</code>
                        </div>
                        <div>
                            <h4 class="text-sm font-medium mb-2">Example: List all zones</h4>
                            <code class="block text-sm bg-gray-100 dark:bg-gray-900 p-3 rounded-lg font-mono overflow-x-auto">curl -H "Authorization: Bearer sdns_your_token_here" http://localhost:8080/api/zones</code>
                        </div>
                    </div>
                </div>
            </main>
        </div>
    </div>

    <!-- Create Token Modal -->
    <div id="createTokenModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
        <div class="bg-white dark:bg-gray-900 rounded-2xl p-6 w-full max-w-md mx-4 shadow-xl">
            <h2 class="text-xl font-bold mb-4">Create API Token</h2>
            <div id="tokenFormSection">
                <div class="mb-4">
                    <label class="block text-sm font-medium mb-2">Token Name</label>
                    <input type="text" id="tokenName" placeholder="My API Token" 
                           class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-brand-500">
                </div>
                <div class="flex gap-3">
                    <button onclick="createAPIToken()" class="flex-1 px-4 py-2.5 bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">Create Token</button>
                    <button onclick="closeTokenModal()" class="px-4 py-2.5 border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">Cancel</button>
                </div>
            </div>
            <div id="tokenResultSection" class="hidden">
                <div class="mb-4 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                    <p class="text-green-700 dark:text-green-400 text-sm mb-2">Token created successfully! Copy it now - you won't be able to see it again.</p>
                </div>
                <div class="mb-4">
                    <label class="block text-sm font-medium mb-2">Your API Token</label>
                    <div class="flex gap-2">
                        <input type="text" id="newTokenValue" readonly
                               class="flex-1 px-4 py-2.5 border border-gray-300 dark:border-gray-700 rounded-lg bg-gray-100 dark:bg-gray-800 font-mono text-sm">
                        <button onclick="copyToken()" class="px-4 py-2.5 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors" title="Copy">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                            </svg>
                        </button>
                    </div>
                </div>
                <button onclick="closeTokenModalAndReload()" class="w-full px-4 py-2.5 bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">Done</button>
            </div>
        </div>
    </div>

    <script>
        function showCreateTokenModal() {
            document.getElementById('createTokenModal').classList.remove('hidden');
            document.getElementById('createTokenModal').classList.add('flex');
            document.getElementById('tokenFormSection').classList.remove('hidden');
            document.getElementById('tokenResultSection').classList.add('hidden');
            document.getElementById('tokenName').value = '';
            document.getElementById('tokenName').focus();
        }

        function closeTokenModal() {
            document.getElementById('createTokenModal').classList.add('hidden');
            document.getElementById('createTokenModal').classList.remove('flex');
        }

        function closeTokenModalAndReload() {
            closeTokenModal();
            window.location.reload();
        }

        async function createAPIToken() {
            const name = document.getElementById('tokenName').value || 'API Token';
            try {
                const resp = await fetch('/account/tokens', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                    body: 'token_name=' + encodeURIComponent(name)
                });
                if (resp.ok) {
                    const data = await resp.json();
                    document.getElementById('newTokenValue').value = data.token;
                    document.getElementById('tokenFormSection').classList.add('hidden');
                    document.getElementById('tokenResultSection').classList.remove('hidden');
                } else {
                    const err = await resp.json();
                    alert('Failed to create token: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }

        function copyToken() {
            const tokenInput = document.getElementById('newTokenValue');
            tokenInput.select();
            document.execCommand('copy');
            alert('Token copied to clipboard!');
        }

        async function deleteAPIToken(id) {
            if (!confirm('Are you sure you want to delete this token? This action cannot be undone.')) return;
            try {
                const resp = await fetch('/account/tokens/' + id, { method: 'DELETE' });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    alert('Failed to delete token');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
` + configModalHTML + `
</body>
</html>
`

// Setup page template - First run admin password creation
const setupHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <title>SimpleDNS - Initial Setup</title>
` + headHTML + `
</head>
<body x-data="{ darkMode: localStorage.getItem('darkMode') === 'true', showPassword: false }" 
      x-init="$watch('darkMode', val => { localStorage.setItem('darkMode', val); document.documentElement.classList.toggle('dark', val) }); document.documentElement.classList.toggle('dark', darkMode)"
      class="bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-white/90 font-sans min-h-screen flex items-center justify-center">
    
    <div class="w-full max-w-md px-6">
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-xl border border-gray-200 dark:border-gray-700 p-8">
            <!-- Logo -->
            <div class="text-center mb-8">
                <div class="flex items-center justify-center gap-3 mb-4">
                    <span class="text-4xl">🌐</span>
                    <span class="text-2xl font-bold">SimpleDNS</span>
                </div>
                <h1 class="text-xl font-semibold mb-2">Welcome!</h1>
                <p class="text-gray-500 dark:text-gray-400">Create your admin password to get started</p>
            </div>

            {{if .Error}}
            <div class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                <p class="text-red-600 dark:text-red-400 text-sm text-center">{{.Error}}</p>
            </div>
            {{end}}

            <form method="POST" action="/setup" class="space-y-6">
                <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Username</label>
                    <input type="text" value="admin" disabled
                           class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed">
                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">The admin username cannot be changed</p>
                </div>

                <div>
                    <label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Password</label>
                    <div class="relative">
                        <input :type="showPassword ? 'text' : 'password'" id="password" name="password" 
                               class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-brand-500 focus:border-transparent outline-none transition pr-12"
                               required autofocus minlength="8">
                        <button type="button" @click="showPassword = !showPassword" 
                                class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
                            <svg x-show="!showPassword" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                            </svg>
                            <svg x-show="showPassword" x-cloak class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
                            </svg>
                        </button>
                    </div>
                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Minimum 8 characters</p>
                </div>

                <div>
                    <label for="confirm_password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Confirm Password</label>
                    <input :type="showPassword ? 'text' : 'password'" id="confirm_password" name="confirm_password" 
                           class="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-brand-500 focus:border-transparent outline-none transition"
                           required minlength="8">
                </div>

                <button type="submit" 
                        class="w-full py-3 px-4 bg-brand-600 hover:bg-brand-700 text-white font-medium rounded-lg transition focus:ring-4 focus:ring-brand-300 dark:focus:ring-brand-800">
                    Create Admin Account
                </button>
            </form>
        </div>

        <!-- Dark mode toggle -->
        <div class="mt-6 text-center">
            <button @click="darkMode = !darkMode" class="text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition">
                <span x-show="!darkMode" class="flex items-center gap-2">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
                    </svg>
                    <span class="text-sm">Dark Mode</span>
                </span>
                <span x-show="darkMode" x-cloak class="flex items-center gap-2">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/>
                    </svg>
                    <span class="text-sm">Light Mode</span>
                </span>
            </button>
        </div>
    </div>
</body>
</html>
`

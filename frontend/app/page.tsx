"use client";

import { useEffect, useState, type KeyboardEvent } from "react";
import { useDebounce } from "../hooks/useDebounce";

// The same struct from your Go backend!
interface Paper {
  arxiv_id: string;
  category?: string;
  title: string;
  abstract: string;
  efficiency_score: number;
  speedup: string;
  hardware: string;
  is_implementable: boolean;
  github_url: string | null;
  edge_type?: string;
  is_theoretical?: boolean;
}

export default function Dashboard() {
  const [selectedPaper, setSelectedPaper] = useState<Paper | null>(null);
  const [interrogationQuery, setInterrogationQuery] = useState("");
  const [isInterrogating, setIsInterrogating] = useState(false);
  type ChatMessage = { role: "user" | "sys_op"; text: string };
  const [chatHistory, setChatHistory] = useState<ChatMessage[]>([]);
  const [papers, setPapers] = useState<Paper[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastSync, setLastSync] = useState("--:--:--");
  const [searchQuery, setSearchQuery] = useState("");
  const debouncedSearchQuery = useDebounce(searchQuery, 500);
  const [apiError, setApiError] = useState<string | null>(null);
  const [activeCategory, setActiveCategory] = useState("ALL");
  const categories = ["ALL", "cs.AI", "cs.LG", "cs.CL", "cs.CV"];

  const parsePapersResponse = async (res: Response): Promise<Paper[]> => {
    const body = await res.text();

    if (!res.ok) {
      throw new Error(body || `Request failed with status ${res.status}`);
    }

    if (!body) {
      return [];
    }

    try {
      const parsed = JSON.parse(body) as unknown;
      return Array.isArray(parsed) ? (parsed as Paper[]) : [];
    } catch {
      throw new Error(`Server returned non-JSON response: ${body.slice(0, 120)}`);
    }
  };

  const fetchLatest = (showLoader = true) => {
    if (showLoader) {
      setLoading(true);
    }
    setApiError(null);

    fetch("http://localhost:8080/api/papers")
      .then(parsePapersResponse)
      .then((data) => {
        setPapers(data || []);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to fetch papers:", err);
        setApiError(err instanceof Error ? err.message : "Failed to fetch papers");
        setPapers([]);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      });
  };

  useEffect(() => {
    fetch("http://localhost:8080/api/papers")
      .then(parsePapersResponse)
      .then((data) => {
        setPapers(data || []);
        setApiError(null);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to fetch papers:", err);
        setApiError(err instanceof Error ? err.message : "Failed to fetch papers");
        setPapers([]);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      });
  }, []);

  const handleSearch = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") {
      return;
    }

    if (searchQuery.trim() === "") {
      fetchLatest();
      return;
    }

    setLoading(true);
    setApiError(null);
    fetch(`http://localhost:8080/api/search?q=${encodeURIComponent(searchQuery)}`)
      .then(parsePapersResponse)
      .then((data) => {
        setPapers(data || []);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to search papers:", err);
        setApiError(err instanceof Error ? err.message : "Failed to search papers");
        setPapers([]);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      });
  };

  const getBadge = (edgeType: string | undefined, score: number): string => {
    const normalized = (edgeType || "").trim();
    const upper = normalized.toUpperCase();

    if (
      upper.includes("OPTIMIZED") ||
      upper.includes("FOCUSED") ||
      upper.includes("MENTIONS")
    ) {
      return upper;
    }

    if (!normalized || normalized.toLowerCase() === "none" || score < 40) {
      return "NOT EFFICIENCY FOCUSED";
    }

    return `${normalized.toUpperCase()} OPTIMIZED`;
  };

  const getScoreBadgeColors = (score: number): string => {
    if (score >= 80) {
      return "bg-[#001a08] text-[#00ff41] border-[#00ff41]/40";
    }
    if (score >= 60) {
      return "bg-[#1f1700] text-[#fbbf24] border-[#fbbf24]/40";
    }
    if (score >= 40) {
      return "bg-[#1f0f00] text-[#f97316] border-[#f97316]/40";
    }
    return "bg-[#1f0707] text-[#ef4444] border-[#ef4444]/40";
  };

  const getPapersWithCodeUrl = (title: string): string =>
    `https://paperswithcode.com/search?q=${encodeURIComponent(title)}`;

  // NEW: Client-side filter
  const displayedPapers = activeCategory === "ALL"
    ? papers
    : papers.filter(p => p.category === activeCategory);

  return (
    <main className="min-h-screen bg-black text-green-500 font-mono pb-16">
      {/* Top Header */}
      <header className="p-4 border-b border-green-900/50 bg-black sticky top-0 z-10 shadow-[0_4px_20px_rgba(0,0,0,0.8)] flex justify-between items-end flex-wrap gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tighter flex items-center gap-2">
            <span className="w-2 h-2 bg-red-500 rounded-full animate-pulse"></span>
            RESEARCH_MONITOR_v1.0
          </h1>
          <p className="text-green-700 text-xs mt-1">
            STATUS: ACTIVE | ENGINE: GO_ROUTER | LLM: GEMMA-4B
          </p>
        </div>

        <div className="flex-1 min-w-[300px] max-w-xl mx-4">
          <div className="relative flex items-center bg-black border border-green-900 focus-within:border-green-400 focus-within:shadow-[0_0_15px_rgba(34,197,94,0.2)] transition-all">
            <span className="pl-3 text-green-500 font-bold">&gt;</span>
            <input
              type="text"
              className="w-full bg-transparent p-2 text-green-400 outline-none placeholder-green-900 text-sm"
              placeholder="EXECUTE_QUERY: (e.g., 'low memory LLM inference')"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={handleSearch}
            />
          </div>
        </div>

        <div className="text-right text-xs text-green-700">
          <p>TOTAL_NODES: {displayedPapers.length}</p>
          <p>LAST_SYNC: {lastSync}</p>
        </div>
      </header>

      {/* NEW CYBERPUNK FILTER BAR */}
      <div className="px-4 py-2 mb-1">
        <div className="flex flex-wrap gap-1.5 text-xs font-semibold border border-green-900/40 bg-[#050505] p-2 rounded-sm">
          {categories.map((cat) => (
            <button
              key={cat}
              onClick={() => setActiveCategory(cat)}
              className={`px-2 py-0.5 text-[11px] transition-all duration-200 ${activeCategory === cat
                ? "bg-green-500 text-black shadow-[0_0_10px_rgba(34,197,94,0.5)]"
                : "text-green-700 hover:text-green-400 hover:bg-green-900/20"
                }`}
            >
              [ {cat} ]
            </button>
          ))}
        </div>
      </div>

      {/* Main Grid */}
      <div className="p-4">
        {apiError && (
          <div className="mb-4 border border-red-900/40 bg-[#120404] p-3 text-xs text-red-300">
            [ API_ERROR ] {apiError}
          </div>
        )}

        {loading ? (
          <div className="text-sm animate-pulse">&gt; Initializing databanks...</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
            {displayedPapers.length === 0 ? (
              <div className="col-span-full flex flex-col items-center justify-center py-24 text-red-500/80 border border-red-900/30 bg-[#050000] shadow-[inset_0_0_20px_rgba(220,38,38,0.1)]">
                <span className="text-4xl mb-4 animate-pulse">_</span>
                <p className="text-sm font-bold tracking-widest">[ ERR: 0_NODES_FOUND ]</p>
                <p className="text-xs mt-2 text-red-700">QUERY EXCEEDS MATHEMATICAL THRESHOLD. NO RELEVANT INTEL EXISTS.</p>
                <button
                  onClick={() => fetchLatest()}
                  className="mt-6 px-4 py-2 border border-red-900/50 text-[10px] hover:bg-red-900/20 hover:text-red-400 transition-colors"
                >
                  [ RESET_DATABANKS ]
                </button>
              </div>
            ) : (
              displayedPapers.map((paper) => {
                const edgeType = (paper.edge_type || "general").toUpperCase();
                const isTheoretical = Boolean(paper.is_theoretical);
                const badge = getBadge(paper.edge_type, paper.efficiency_score);
                const scoreBadgeColors = getScoreBadgeColors(paper.efficiency_score);

                return (
                  <div
                    key={paper.arxiv_id}
                    onClick={() => setSelectedPaper(paper)}
                    className="relative flex flex-col border border-green-900/40 bg-[#050505] p-3 hover:border-green-500 hover:bg-[#0a0a0a] transition-all duration-200"
                  >
                    {/* The "Edge" Ribbon */}
                    <div className={`absolute top-0 right-0 text-[8px] px-1 uppercase tracking-widest border-l border-b ${scoreBadgeColors}`}>
                      {badge}
                    </div>

                    {/* ID & Score */}
                    <div className="flex justify-between items-start mb-2 pb-2 border-b border-green-900/30">
                      <span className="text-[10px] text-green-800 truncate pr-2" title={paper.arxiv_id}>
                        {paper.arxiv_id.split("v")[0]}
                      </span>
                      <span
                        className={`text-[10px] font-bold px-1.5 py-0.5 border ${scoreBadgeColors}`}
                      >
                        SCORE:{paper.efficiency_score}
                      </span>
                    </div>

                    {/* Title */}
                    <h2 className="text-xs font-bold mb-3 line-clamp-3 leading-relaxed text-gray-200">
                      {paper.title}
                    </h2>

                    {/* Metrics */}
                    <div className="text-[10px] space-y-1.5 mt-auto bg-black/50 p-2 border border-green-900/20">
                      <div className="flex justify-between">
                        <span className="text-green-800">SPEEDUP:</span>
                        <span className="text-gray-400 text-right truncate w-24">{paper.speedup || "N/A"}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-green-800">HARDWARE:</span>
                        <span className="text-gray-400 text-right truncate w-24">{paper.hardware || "N/A"}</span>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-2 mt-2 text-[10px]">
                      <div
                        className={`p-1 border ${isTheoretical ? "border-red-900/30 text-red-900" : "border-green-900/30 text-green-500"
                          }`}
                      >
                        STATUS: {isTheoretical ? "THEORETICAL" : "EMPIRICAL"}
                      </div>
                      <div className="p-1 border border-green-900/30 text-green-500">EDGE: {edgeType}</div>
                    </div>

                    {/* Cyberpunk Button */}
                    {paper.github_url ? (
                      <a
                        href={paper.github_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="mt-3 block w-full text-center text-[10px] font-bold py-1.5 bg-green-900/20 text-green-400 border border-green-500/50 hover:bg-green-500 hover:text-black hover:shadow-[0_0_10px_rgba(34,197,94,0.4)] transition-all"
                      >
                        [&lt;/&gt; GET_CODE]
                      </a>
                    ) : (
                      <a
                        href={getPapersWithCodeUrl(paper.title)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="mt-3 block w-full text-center text-[10px] font-bold py-1.5 bg-green-900/20 text-green-400 border border-green-500/50 hover:bg-green-500 hover:text-black hover:shadow-[0_0_10px_rgba(34,197,94,0.4)] transition-all"
                      >
                        [ SEARCH PAPERS WITH CODE ]
                      </a>
                    )}
                  </div>
                );
              })
            )}
          </div>
        )}
      </div>

      {/* Live Ticker Footer */}
      <footer className="fixed bottom-0 left-0 w-full bg-black border-t border-green-900 py-1 overflow-hidden z-20">
        <div className="animate-ticker text-[10px] text-green-500 font-bold tracking-widest flex gap-8">
          <span>&gt; LATEST INTEL: </span>
          {displayedPapers.filter(p => p.efficiency_score > 50).map(p => (
            <span key={`ticker-${p.arxiv_id}`}>
              [{p.speedup && p.speedup !== "None" ? `SPEEDUP: ${p.speedup} | ` : ""}{p.title.toUpperCase()}] ///
            </span>
          ))}
        </div>
      </footer>

      {/* THE INTERROGATE NODE DRAWER */}
      {selectedPaper && (
        <div className="fixed top-0 right-0 h-full w-full md:w-[400px] bg-black border-l border-green-500 shadow-[-10px_0_30px_rgba(34,197,94,0.1)] z-50 flex flex-col font-mono transform transition-transform duration-300">

          {/* Header */}
          <div className="p-4 border-b border-green-900/50 flex justify-between items-start bg-[#050505]">
            <div>
              <p className="text-red-500 text-xs animate-pulse">_SYSTEM_OVERRIDE</p>
              <h3 className="text-green-500 text-sm font-bold mt-1">INTERROGATE_NODE</h3>
              <p className="text-green-800 text-[10px] truncate max-w-[250px]">{selectedPaper.title}</p>
            </div>
            <button
              onClick={() => {
                setSelectedPaper(null);
                setChatHistory([]);
                setInterrogationQuery("");
              }}
              className="text-green-500 hover:text-red-500 font-bold"
            >
              [ X ]
            </button>
          </div>

          {/* Chat Window */}
          {/* Chat Window */}
          <div className="flex-1 p-4 overflow-y-auto bg-black flex flex-col gap-4 text-xs font-mono">
            <div className="text-green-700 mb-4 border-b border-green-900/30 pb-2">
              &gt; ESTABLISHING NEURAL LINK TO DOCUMENT... <br />
              &gt; NODE READY. <br />
              &gt; WHAT DO YOU WANT TO KNOW?
            </div>

            {/* Loop through the history and render the conversation */}
            {chatHistory.map((msg, idx) => (
              <div
                key={idx}
                className={`p-3 border ${msg.role === "user"
                  ? "border-green-900/30 bg-[#0a0a0a] text-green-300 ml-4" // User messages indented right
                  : "border-red-900/30 bg-[#050000] text-red-400 mr-4"     // AI messages indented left
                  }`}
              >
                <span className={`font-bold ${msg.role === "user" ? "text-green-500" : "text-red-500"}`}>
                  {msg.role === "user" ? "YOU: " : "SYS_OP: "}
                </span>
                {msg.text}
              </div>
            ))}

            {/* Loading state indicator */}
            {isInterrogating && (
              <div className="text-red-500 animate-pulse mt-2">&gt; ANALYZING VECTORS...</div>
            )}
          </div>

          {/* Input Area */}
          <div className="p-4 border-t border-green-900/50 bg-[#050505]">
            <div className="flex gap-2">
              <input
                type="text"
                value={interrogationQuery}
                onChange={(e) => setInterrogationQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && interrogationQuery.trim() !== "") {
                    // 1. Save user query and immediately add to UI history
                    const userText = interrogationQuery;
                    setChatHistory(prev => [...prev, { role: "user", text: userText }]);
                    setInterrogationQuery(""); // Clear input box
                    setIsInterrogating(true);

                    // 2. Fire to your existing Go API
                    fetch("http://localhost:8080/api/interrogate", {
                      method: "POST",
                      headers: { "Content-Type": "application/json" },
                      body: JSON.stringify({ paper_id: selectedPaper.arxiv_id, question: userText })
                    })
                      .then(async (res) => {
                        if (!res.ok) throw new Error("NODE UNRESPONSIVE");
                        return res.json();
                      })
                      .then(data => {
                        // 3. Append the AI's answer to the UI history
                        setChatHistory(prev => [...prev, { role: "sys_op", text: data.answer || data.error }]);
                        setIsInterrogating(false);
                      })
                      .catch(err => {
                        setChatHistory(prev => [...prev, { role: "sys_op", text: `[ERR] ${err.message}` }]);
                        setIsInterrogating(false);
                      });
                  }
                }}
                disabled={isInterrogating}
                className="flex-1 bg-black border border-green-900 p-2 text-green-500 outline-none focus:border-green-500 placeholder-green-900 text-xs"
                placeholder="Ask about hardware, speedups, limits..."
              />
            </div>
          </div>

        </div>
      )}
    </main>
  );
}
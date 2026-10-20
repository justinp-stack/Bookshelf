"use client";
import { useState, useEffect } from "react";
import { NextResponse } from 'next/server';
import Link from 'next/link'; 

export async function GET() {
  return NextResponse.json(
    { 
      status: 'UP',
      timestamp: new Date().toISOString()
    },
    { status: 200 }
  );
}

interface BookType {
  id: number;
  title: string;
  AuthorID?: string; 
  GenreID?: string;  
  pages: number;
  PublisherID?: string;
}

interface EntityType {
  id: string;
  name: string;
}

export default function Page() {
  const [token, setToken] = useState<string>("");
  const [isAuthReady, setIsAuthReady] = useState(false);

  const [books, setBooks] = useState<BookType[]>([]);
  const [publishers, setPublishers] = useState<EntityType[]>([]); 
  const [authors, setAuthors] = useState<EntityType[]>([]); 
  const [genres, setGenres] = useState<EntityType[]>([]); 
  
  const [sortBy, setSortBy] = useState("title"); 
  const [hasError, setHasError] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const [isFormVisible, setIsFormVisible] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [successMessage, setSuccessMessage] = useState("");
  const [newTitle, setNewTitle] = useState("");
  const [newPages, setNewPages] = useState<number | "">(300);
  
  const [newPublisherId, setNewPublisherId] = useState("");
  const [newAuthorId, setNewAuthorId] = useState("");
  const [newGenreId, setNewGenreId] = useState("");

  const [isCreatingPub, setIsCreatingPub] = useState(false);
  const [newPubName, setNewPubName] = useState("");
  const [isSavingPub, setIsSavingPub] = useState(false);

  const [isCreatingAuthor, setIsCreatingAuthor] = useState(false);
  const [newAuthorName, setNewAuthorName] = useState("");
  const [isSavingAuthor, setIsSavingAuthor] = useState(false);

  const [isCreatingGenre, setIsCreatingGenre] = useState(false);
  const [newGenreName, setNewGenreName] = useState("");
  const [isSavingGenre, setIsSavingGenre] = useState(false);

  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    const initializeAuth = async () => {
      let storedToken = localStorage.getItem("my_app_jwt_token");
      if (!storedToken) {
        try {
          const res = await fetch("http://localhost:8080/token", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ client_name: "NextJS Web Client" })
          });
          
          if (res.ok) {
            const data = await res.json();
            storedToken = data.token ?? null;
            if (storedToken) {
              localStorage.setItem("my_app_jwt_token", storedToken);
            }
          } else {
            console.error("Backend rejected token generation");
          }
        } catch (err) {
          console.error("Cannot reach the Go backend to get token", err);
        }
      }
      
      if (storedToken) {
        setToken(storedToken);
      }
      setIsAuthReady(true);
    };

    initializeAuth();
  }, []);

  const fetchData = async (currentToken: string) => {
    try {
      const headers = { 'Authorization': `Bearer ${currentToken}` };

      const [booksRes, pubRes, authRes, genRes] = await Promise.all([
        fetch('http://localhost:8080/books', { headers }),
        fetch('http://localhost:8080/publishers', { headers }),
        fetch('http://localhost:8080/authors', { headers }),
        fetch('http://localhost:8080/genres', { headers })
      ]);

      if (!booksRes.ok) {
        if (booksRes.status === 401) {
          localStorage.removeItem("my_app_jwt_token");
        }
        throw new Error(`Failed to fetch books: ${booksRes.status}`);
      }

      const booksData = await booksRes.json();
      setBooks(Array.isArray(booksData) ? booksData : (booksData.books || []));

      if (pubRes.ok) setPublishers(await pubRes.json() || []);
      if (authRes.ok) setAuthors(await authRes.json() || []);
      if (genRes.ok) setGenres(await genRes.json() || []);

    } catch (error) {
      console.error("Error fetching data:", error);
      setHasError(true);
    } finally {
      setIsLoading(false);
    }
  };
  
  useEffect(() => {
    if (isAuthReady) {
      if (token) {
        fetchData(token);
      } else {
        setHasError(true);
        setIsLoading(false);
      }
    }
  }, [isAuthReady, token]);

  const handleCreateAuthor = async (e: React.MouseEvent) => {
    e.preventDefault();
    if (!newAuthorName.trim()) {
      setErrors(prev => ({ ...prev, author: "Author name cannot be empty." }));
      return;
    }
    setErrors(prev => {
      const newErrors = { ...prev };
      delete newErrors.author;
      return newErrors;
    });
    
    setIsSavingAuthor(true);
    try {
      const response = await fetch('http://localhost:8080/authors', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}` 
        },
        body: JSON.stringify({ name: newAuthorName }),
      });
      if (!response.ok) throw new Error("Failed to add author");
      const savedAuth = await response.json();
      setAuthors([...authors, savedAuth]);
      setNewAuthorId(savedAuth.id);
      setNewAuthorName("");
      setIsCreatingAuthor(false);
    } catch (error) {
      alert("Failed to create author.");
    } finally {
      setIsSavingAuthor(false);
    }
  };

  const handleCreateGenre = async (e: React.MouseEvent) => {
    e.preventDefault();
    if (!newGenreName.trim()) {
      setErrors(prev => ({ ...prev, genre: "Genre name cannot be empty." }));
      return;
    }
    setErrors(prev => {
      const newErrors = { ...prev };
      delete newErrors.genre;
      return newErrors;
    });

    setIsSavingGenre(true);
    try {
      const response = await fetch('http://localhost:8080/genres', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ name: newGenreName }),
      });
      if (!response.ok) throw new Error("Failed to add genre");
      const savedGenre = await response.json();
      setGenres([...genres, savedGenre]);
      setNewGenreId(savedGenre.id);
      setNewGenreName("");
      setIsCreatingGenre(false);
    } catch (error) {
      alert("Failed to create genre.");
    } finally {
      setIsSavingGenre(false);
    }
  };

  const handleCreatePublisher = async (e: React.MouseEvent) => {
    e.preventDefault();
    if (!newPubName.trim()) {
      setErrors(prev => ({ ...prev, publisher: "Publisher name cannot be empty." }));
      return;
    }
    setErrors(prev => {
      const newErrors = { ...prev };
      delete newErrors.publisher;
      return newErrors;
    });

    setIsSavingPub(true);
    try {
      const response = await fetch('http://localhost:8080/publishers', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ name: newPubName }),
      });
      if (!response.ok) throw new Error("Failed to add publisher");
      const savedPub = await response.json();
      setPublishers([...publishers, savedPub]);
      setNewPublisherId(savedPub.id);
      setNewPubName("");
      setIsCreatingPub(false);
    } catch (error) {
      alert("Failed to create publisher.");
    } finally {
      setIsSavingPub(false);
    }
  };

  const handleAddBook = async (e: React.FormEvent) => {
    e.preventDefault(); 

    const newErrors: Record<string, string> = {};
    if (!newTitle.trim()) {
      newErrors.title = "Title cannot be empty.";
    }
    if (newPages === "" || Number(newPages) < 1) {
      newErrors.pages = "Page number must be 1 or greater.";
    }
    
    if (isCreatingAuthor && !newAuthorName.trim()) {
      newErrors.author = "Author name cannot be empty.";
    } else if (!isCreatingAuthor && !newAuthorId) {
      newErrors.author = "Please select or create an author.";
    }
    
    if (isCreatingGenre && !newGenreName.trim()) {
      newErrors.genre = "Genre name cannot be empty.";
    } else if (!isCreatingGenre && !newGenreId) {
       newErrors.genre = "Please select or create a genre.";
    }

    if (isCreatingPub && !newPubName.trim()) {
      newErrors.publisher = "Publisher name cannot be empty.";
    } else if (!isCreatingPub && !newPublisherId) {
      newErrors.publisher = "Please select or create a publisher.";
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return; 
    }
    
    setErrors({}); 
    setIsSubmitting(true);

    const bookPayload = {
      title: newTitle,
      pages: Number(newPages),
      AuthorID: newAuthorId || undefined,
      author: isCreatingAuthor ? newAuthorName : undefined,
      GenreID: newGenreId || undefined,
      genre: isCreatingGenre ? newGenreName : undefined,
      PublisherID: newPublisherId || undefined, 
      publisher: isCreatingPub ? newPubName : undefined,
    };

    try {
      const response = await fetch('http://localhost:8080/books', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(bookPayload),
      });

      if (!response.ok) throw new Error("Failed to add book");

      await fetchData(token);

      setNewTitle("");
      setNewPages(300);
      setNewAuthorId("");
      setNewAuthorName("");
      setNewGenreId("");
      setNewGenreName("");
      setNewPublisherId("");
      setNewPubName("");
      setIsCreatingAuthor(false);
      setIsCreatingGenre(false);
      setIsCreatingPub(false);
      setIsFormVisible(false);

      setSuccessMessage("Book successfully added to catalog!");
      setTimeout(() => {
        setSuccessMessage("");
      }, 3000);

    } catch (error) {
      console.error("Error adding book:", error);
      alert("Failed to add book.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const booksToDisplay = [...books];
  
  booksToDisplay.sort((a, b) => {
    if (sortBy === "title") {
      return (a.title || "").toLowerCase().localeCompare((b.title || "").toLowerCase());
    }
    if (sortBy === "author") {
      const aName = authors.find(x => String(x.id) === String(a.AuthorID))?.name || a.AuthorID || "";
      const bName = authors.find(x => String(x.id) === String(b.AuthorID))?.name || b.AuthorID || "";
      return aName.toLowerCase().localeCompare(bName.toLowerCase());
    }
    if (sortBy === "genre") {
      const aName = genres.find(x => String(x.id) === String(a.GenreID))?.name || a.GenreID || "";
      const bName = genres.find(x => String(x.id) === String(b.GenreID))?.name || b.GenreID || "";
      return aName.toLowerCase().localeCompare(bName.toLowerCase());
    }
    return 0;
  });

  const MAX_SHELF_WIDTH = 550;
  const GAP_WIDTH = 0;
  
  const getBookPixelWidth = (pages = 300) => {
    if (pages < 50 ) return 16;
    if (pages < 150) return 24;
    if (pages < 300) return 32;
    if (pages < 500) return 40;
    if (pages < 800) return 48;
    return 56;
  };

  const shelves: BookType[][] = [];
  let currentShelf: BookType[] = [];
  let currentShelfWidth = 0;

  for (const book of booksToDisplay) {
    const bookWidth = getBookPixelWidth(book.pages);
    const widthToAdd = currentShelf.length === 0 ? bookWidth : bookWidth + GAP_WIDTH;
    
    if (currentShelfWidth + widthToAdd > MAX_SHELF_WIDTH) {
      shelves.push(currentShelf);
      currentShelf = [book];
      currentShelfWidth = bookWidth;
    } else {
      currentShelf.push(book);
      currentShelfWidth += widthToAdd;
    }
  }
  
  if (currentShelf.length > 0) {
    shelves.push(currentShelf);
  }

  if (isLoading) {
    return <div className="text-center mt-10">Authenticating and loading books...</div>;
  }

  if (hasError) {
    return (
      <div className="text-center mt-10">
        <p className="text-red-500 mb-4">Failed to load data. Is the Go backend running on port 8080?</p>
        <button 
          onClick={() => {
            localStorage.removeItem("my_app_jwt_token");
            window.location.reload();
          }}
          className="bg-gray-800 text-white px-4 py-2 rounded"
        >
          Reset Auth Key & Reload
        </button>
      </div>
    );
  }

  return (
    <div className="h-screen w-full pt-10 overflow-y-auto bg-gray-100">
      <div className="flex flex-col items-center">

        {/* Top Navigation & Action Buttons */}
        <div className="mb-8 flex items-center gap-6">
          <button 
            onClick={() => setIsFormVisible(!isFormVisible)}
            className={`font-black uppercase tracking-widest text-sm py-3 px-6 border-[3px] border-black transition-all shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] active:shadow-none active:translate-x-1 active:translate-y-1 ${
              isFormVisible 
                ? "bg-gray-200 text-black hover:bg-gray-300" 
                : "bg-white text-black hover:bg-gray-50"
            }`}
          >
            {isFormVisible ? "CLOSE LEDGER" : "+ NEW BOOK"}
          </button>

          <Link 
            href="/sign-up"
            className="bg-white text-black font-black uppercase tracking-widest text-sm py-3 px-6 border-[3px] border-black transition-all shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] hover:bg-gray-50 active:shadow-none active:translate-x-1 active:translate-y-1"
          >
          BookWorm SIGN UP
          </Link>

          <Link 
            href="/check-out"
            className="flex items-center gap-3 bg-black text-white font-black uppercase tracking-widest text-sm py-3 px-6 border-[3px] border-black transition-all shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] hover:bg-gray-800 active:shadow-none active:translate-x-1 active:translate-y-1"
            title="Go to Checkout Desk"
          >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="square" strokeLinejoin="miter" className="w-5 h-5">
              <path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20" />
              <circle cx="12" cy="8" r="2" />
              <path d="M15 13a3 3 0 1 0-6 0" />
            </svg>
            CHECKOUT DESK
          </Link>
        </div>

        {successMessage && (
          <div className="mb-8 border-[3px] border-black bg-white p-4 w-[700px] text-center shadow-[6px_6px_0px_0px_rgba(0,0,0,1)]">
            <span className="font-mono text-lg font-bold uppercase tracking-widest text-black">
              ✓ {successMessage}
            </span>
          </div>
        )}

        {/* --- The Leather Form --- */}
        {isFormVisible && (
          <form 
            onSubmit={handleAddBook} 
            className="mb-12 flex flex-col gap-6 bg-[#362114] p-10 rounded-md shadow-[0_20px_40px_rgba(0,0,0,0.6)] w-[700px] relative border-[6px] border-[#22140c]"
            noValidate
          >
            {/* Simulated Leather Stitching */}
            <div className="absolute inset-2 border-[1.5px] border-dashed border-[#543622] pointer-events-none rounded-sm"></div>

            <h2 className="text-3xl font-serif text-center uppercase tracking-widest text-[#e8dcc4] border-b-[2px] border-[#543622] pb-5 drop-shadow-[0_2px_2px_rgba(0,0,0,0.8)]">
              New Book Form
            </h2>
            
            <div className="grid grid-cols-3 gap-6 relative z-10">
              <div className="col-span-2 flex flex-col gap-2">
                <label className="text-xs font-serif font-bold uppercase tracking-widest text-[#b5a397]">Title</label>
                <input 
                  type="text" 
                  placeholder="e.g. The Hobbit" 
                  value={newTitle} 
                  onChange={(e) => {
                    setNewTitle(e.target.value);
                    if (errors.title) setErrors(prev => ({...prev, title: ""}));
                  }} 
                  className={`w-full bg-[#fdf8ef] border-2 outline-none px-4 py-3 text-lg font-serif text-[#362114] focus:border-[#8b7355] transition-colors rounded-sm shadow-inner ${errors.title ? 'border-red-800' : 'border-[#8b7355]'}`}
                />
                {errors.title && <span className="text-red-400 font-bold text-xs uppercase tracking-widest">{errors.title}</span>}
              </div>
              <div className="flex flex-col gap-2">
                <label className="text-xs font-serif font-bold uppercase tracking-widest text-[#b5a397]">Pages</label>
                <input 
                  type="number" 
                  min="1"
                  value={newPages} 
                  onChange={(e) => {
                    setNewPages(e.target.value === "" ? "" : Number(e.target.value));
                    if (errors.pages) setErrors(prev => ({...prev, pages: ""}));
                  }} 
                  className={`w-full bg-[#fdf8ef] border-2 outline-none px-4 py-3 text-lg font-serif text-[#362114] focus:border-[#8b7355] transition-colors rounded-sm shadow-inner ${errors.pages ? 'border-red-800' : 'border-[#8b7355]'}`}
                />
                {errors.pages && <span className="text-red-400 font-bold text-xs uppercase tracking-widest">{errors.pages}</span>}
              </div>
            </div>

            <div className="flex flex-col gap-2 relative z-10">
              <label className="text-xs font-serif font-bold uppercase tracking-widest text-[#b5a397]">Author</label>
              {!isCreatingAuthor ? (
                <div className="flex flex-col gap-1">
                  <div className="flex gap-3">
                    <select
                      value={newAuthorId}
                      onChange={(e) => {
                        setNewAuthorId(e.target.value);
                        if (errors.author) setErrors(prev => ({...prev, author: ""}));
                      }}
                      className={`w-full bg-[#fdf8ef] border-2 outline-none px-4 py-3 text-lg font-serif text-[#362114] focus:border-[#8b7355] cursor-pointer rounded-sm shadow-inner ${errors.author ? 'border-red-800' : 'border-[#8b7355]'}`}
                    >
                      <option value="">-- Select an Author --</option>
                      {authors.map((auth) => (
                        <option key={auth.id} value={auth.id}>{auth.name}</option>
                      ))}
                    </select>
                    <button 
                      type="button"
                      onClick={() => {
                        setIsCreatingAuthor(true);
                        setNewAuthorId("");
                        if (errors.author) setErrors(prev => ({...prev, author: ""}));
                      }}
                      className="shrink-0 bg-[#22140c] text-[#e8dcc4] font-serif font-bold uppercase tracking-widest px-6 border border-[#543622] hover:bg-[#1a0f09] transition-colors rounded-sm"
                    >
                      + New
                    </button>
                  </div>
                  {errors.author && <span className="text-red-400 font-bold text-xs uppercase tracking-widest">{errors.author}</span>}
                </div>
              ) : (
                <div className="flex flex-col gap-1">
                  <div className="flex gap-3">
                    <input 
                      type="text" 
                      placeholder="Author Name..." 
                      value={newAuthorName}
                      onChange={(e) => {
                        setNewAuthorName(e.target.value);
                        if (errors.author) setErrors(prev => ({...prev, author: ""}));
                      }}
                      className={`w-full bg-[#fdf8ef] border-2 outline-none px-4 py-3 text-lg font-serif text-[#362114] focus:border-[#8b7355] rounded-sm shadow-inner ${errors.author ? 'border-red-800' : 'border-[#8b7355]'}`}
                      autoFocus
                    />
                    <button 
                      type="button"
                      onClick={handleCreateAuthor}
                      disabled={isSavingAuthor}
                      className="shrink-0 bg-[#543622] text-[#e8dcc4] font-serif font-bold uppercase tracking-widest px-6 border border-[#8b7355] hover:bg-[#6b442b] transition-all rounded-sm disabled:opacity-50"
                    >
                      {isSavingAuthor ? "..." : "Save"}
                    </button>
                    <button 
                      type="button"
                      onClick={() => { 
                        setIsCreatingAuthor(false); 
                        setNewAuthorName(""); 
                        setErrors(prev => {
                          const newErrors = {...prev};
                          delete newErrors.author;
                          return newErrors;
                        }); 
                      }}
                      className="shrink-0 bg-[#362114] text-[#b5a397] font-serif font-bold text-xl px-5 border border-[#543622] hover:bg-[#22140c] hover:text-[#e8dcc4] transition-colors rounded-sm"
                    >
                      X
                    </button>
                  </div>
                  {errors.author && <span className="text-red-400 font-bold text-xs uppercase tracking-widest">{errors.author}</span>}
                </div>
              )}
            </div>

            <div className="flex flex-col gap-2 relative z-10">
              <label className="text-xs font-serif font-bold uppercase tracking-widest text-[#b5a397]">Genre</label>
              {!isCreatingGenre ? (
                <div className="flex flex-col gap-1">
                  <div className="flex gap-3">
                    <select
                      value={newGenreId}
                      onChange={(e) => {
                        setNewGenreId(e.target.value);
                        if (errors.genre) setErrors(prev => ({...prev, genre: ""}));
                      }}
                      className={`w-full bg-[#fdf8ef] border-2 outline-none px-4 py-3 text-lg font-serif text-[#362114] focus:border-[#8b7355] cursor-pointer rounded-sm shadow-inner ${errors.genre ? 'border-red-800' : 'border-[#8b7355]'}`}
                    >
                      <option value="">-- Select a Genre --</option>
                      {genres.map((gen) => (
                        <option key={gen.id} value={gen.id}>{gen.name}</option>
                      ))}
                    </select>
                    <button 
                      type="button"
                      onClick={() => {
                        setIsCreatingGenre(true);
                        setNewGenreId("");
                        if (errors.genre) setErrors(prev => ({...prev, genre: ""}));
                      }}
                      className="shrink-0 bg-[#22140c] text-[#e8dcc4] font-serif font-bold uppercase tracking-widest px-6 border border-[#543622] hover:bg-[#1a0f09] transition-colors rounded-sm"
                    >
                      + New
                    </button>
                  </div>
                  {errors.genre && <span className="text-red-400 font-bold text-xs uppercase tracking-widest">{errors.genre}</span>}
                </div>
              ) : (
                <div className="flex flex-col gap-1">
                  <div className="flex gap-3">
                    <input 
                      type="text" 
                      placeholder="Genre Name..." 
                      value={newGenreName}
                      onChange={(e) => {
                        setNewGenreName(e.target.value);
                        if (errors.genre) setErrors(prev => ({...prev, genre: ""}));
                      }}
                      className={`w-full bg-[#fdf8ef] border-2 outline-none px-4 py-3 text-lg font-serif text-[#362114] focus:border-[#8b7355] rounded-sm shadow-inner ${errors.genre ? 'border-red-800' : 'border-[#8b7355]'}`}
                      autoFocus
                    />
                    <button 
                      type="button"
                      onClick={handleCreateGenre}
                      disabled={isSavingGenre}
                      className="shrink-0 bg-[#543622] text-[#e8dcc4] font-serif font-bold uppercase tracking-widest px-6 border border-[#8b7355] hover:bg-[#6b442b] transition-all rounded-sm disabled:opacity-50"
                    >
                      {isSavingGenre ? "..." : "Save"}
                    </button>
                    <button 
                      type="button"
                      onClick={() => { 
                        setIsCreatingGenre(false); 
                        setNewGenreName(""); 
                        setErrors(prev => {
                          const newErrors = {...prev};
                          delete newErrors.genre;
                          return newErrors;
                        }); 
                      }}
                      className="shrink-0 bg-[#362114] text-[#b5a397] font-serif font-bold text-xl px-5 border border-[#543622] hover:bg-[#22140c] hover:text-[#e8dcc4] transition-colors rounded-sm"
                    >
                      X
                    </button>
                  </div>
                  {errors.genre && <span className="text-red-400 font-bold text-xs uppercase tracking-widest">{errors.genre}</span>}
                </div>
              )}
            </div>

            <div className="flex flex-col gap-2 relative z-10">
              <label className="text-xs font-serif font-bold uppercase tracking-widest text-[#b5a397]">Publisher</label>
              {!isCreatingPub ? (
                <div className="flex flex-col gap-1">
                  <div className="flex gap-3">
                    <select
                      value={newPublisherId}
                      onChange={(e) => {
                        setNewPublisherId(e.target.value);
                        if (errors.publisher) setErrors(prev => ({...prev, publisher: ""}));
                      }}
                      className={`w-full bg-[#fdf8ef] border-2 outline-none px-4 py-3 text-lg font-serif text-[#362114] focus:border-[#8b7355] cursor-pointer rounded-sm shadow-inner ${errors.publisher ? 'border-red-800' : 'border-[#8b7355]'}`}
                    >
                      <option value="">-- Select a Publisher --</option>
                      {publishers.map((pub) => (
                        <option key={pub.id} value={pub.id}>{pub.name}</option>
                      ))}
                    </select>
                    <button 
                      type="button"
                      onClick={() => {
                        setIsCreatingPub(true);
                        setNewPublisherId("");
                        if (errors.publisher) setErrors(prev => ({...prev, publisher: ""}));
                      }}
                      className="shrink-0 bg-[#22140c] text-[#e8dcc4] font-serif font-bold uppercase tracking-widest px-6 border border-[#543622] hover:bg-[#1a0f09] transition-colors rounded-sm"
                    >
                      + New
                    </button>
                  </div>
                  {errors.publisher && <span className="text-red-400 font-bold text-xs uppercase tracking-widest">{errors.publisher}</span>}
                </div>
              ) : (
                <div className="flex flex-col gap-1">
                  <div className="flex gap-3">
                    <input 
                      type="text" 
                      placeholder="Publisher Name..." 
                      value={newPubName}
                      onChange={(e) => {
                        setNewPubName(e.target.value);
                        if (errors.publisher) setErrors(prev => ({...prev, publisher: ""}));
                      }}
                      className={`w-full bg-[#fdf8ef] border-2 outline-none px-4 py-3 text-lg font-serif text-[#362114] focus:border-[#8b7355] rounded-sm shadow-inner ${errors.publisher ? 'border-red-800' : 'border-[#8b7355]'}`}
                      autoFocus
                    />
                    <button 
                      type="button"
                      onClick={handleCreatePublisher}
                      disabled={isSavingPub}
                      className="shrink-0 bg-[#543622] text-[#e8dcc4] font-serif font-bold uppercase tracking-widest px-6 border border-[#8b7355] hover:bg-[#6b442b] transition-all rounded-sm disabled:opacity-50"
                    >
                      {isSavingPub ? "..." : "Save"}
                    </button>
                    <button 
                      type="button"
                      onClick={() => { 
                        setIsCreatingPub(false); 
                        setNewPubName(""); 
                        setErrors(prev => {
                          const newErrors = {...prev};
                          delete newErrors.publisher;
                          return newErrors;
                        }); 
                      }}
                      className="shrink-0 bg-[#362114] text-[#b5a397] font-serif font-bold text-xl px-5 border border-[#543622] hover:bg-[#22140c] hover:text-[#e8dcc4] transition-colors rounded-sm"
                    >
                      X
                    </button>
                  </div>
                  {errors.publisher && <span className="text-red-400 font-bold text-xs uppercase tracking-widest">{errors.publisher}</span>}
                </div>
              )}
            </div>

            <button 
              type="submit" 
              disabled={isSubmitting}
              className="mt-6 w-full bg-[#22140c] text-[#e8dcc4] text-xl uppercase tracking-widest font-serif font-bold py-4 rounded-sm shadow-lg hover:bg-[#1a0f09] transition-all border border-[#543622] active:scale-[0.98] disabled:opacity-50 relative z-10"
            >
              {isSubmitting ? "Embossing Ledger..." : "Add to Shelf"}
            </button>
          </form>
        )}

        {/* --- Bookshelf Display Area --- */}
        <div className="flex items-center gap-3 mb-10 mt-4 border-2 border-black p-2 bg-white">
          <label htmlFor="sortSelect" className="font-bold uppercase tracking-widest text-sm cursor-pointer">
            Sort by:
          </label>
          <select
            id="sortSelect"
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value)}
            className="border-[2px] border-black rounded-none px-3 py-1 cursor-pointer bg-gray-50 text-black font-bold uppercase text-sm outline-none"
          >
            <option value="author">Author</option>
            <option value="title">Title</option>
            <option value="genre">Genre</option>
          </select>
        </div>

        <div className="flex flex-col items-center gap-12 pb-16">
          {shelves.map((shelfBooks, index) => (
            <div key={index} className="flex flex-col items-center">
              <div className="flex items-end justify-start w-[600px] mb-1 gap-1">
                {shelfBooks.map((book: BookType) => {
                  const displayGenre = genres.find(g => String(g.id) === String(book.GenreID))?.name || book.GenreID;
                  
                  return (
                    <Book 
                      key={book.id} 
                      title={book.title} 
                      genre={displayGenre} 
                      pages={book.pages} 
                    />
                  );
                })}
              </div>
              <div className="w-[600px] h-5 bg-black mx-auto rounded-none shadow-[4px_4px_0px_0px_rgba(0,0,0,0.5)] border-b-2 border-gray-800"></div>
            </div>
          ))}
        </div>

      </div>
    </div>
  );
}

function Book({ title, genre, pages = 300 }: { title: string; genre?: string; pages?: number }) {
  const getGenreColor = (genreString?: string) => {
    const genre = genreString ? genreString.toLowerCase() : "";
    
    const colorMap: Record<string, string> = {
      "fantasy": "bg-[#f786e6]",
      "sci-fi": "bg-[#251d54]",
      "science fiction": "bg-[#251d54]",
      "memoir": "bg-[#522c19]",
      "psychological thriller": "bg-[#961818]",
      "dystopian": "bg-[#7f1d1d]",
      "manifesto": "bg-[#ef4444]",
      "parody": "bg-[#f97316]",
      "political affair": "bg-[#115e59]",
      "dieselpunk": "bg-[#57534e]",
      "satire": "bg-[#84cc16]",
      "adventure": "bg-[#0891b2]",
      "romance": "bg-[#ff0000]",
      "fiction": "bg-[#0004ff]", 
    };

    for (const [key, color] of Object.entries(colorMap)) {
      if (genre.includes(key)) {
        return color;
      }
    }

    return "bg-[#2563eb]"; 
  };

  const getThickness = (pageCount: number) => {
    if (pageCount < 150) return "w-6";  
    if (pageCount < 300) return "w-8";  
    if (pageCount < 500) return "w-10"; 
    if (pageCount < 800) return "w-12"; 
    return "w-14";                      
  };

  const colorClasses = getGenreColor(genre);
  const thicknessClass = getThickness(pages);

  return (
    <Link 
      href={`/check-out?title=${encodeURIComponent(title)}`}
      title={`Genre: ${genre} | Pages: ${pages}`} 
      className={`[writing-mode:vertical-lr] rotate-180 inline-flex flex-col items-center justify-center text-white font-semibold py-1.5 px-5 rounded-md tracking-widest uppercase transition-all hover:-translate-y-2 hover:shadow-lg cursor-pointer border-[1.5px] border-black/40 ${colorClasses} ${thicknessClass}`}
    >
      {title}
    </Link>
  );
}
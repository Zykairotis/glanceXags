from fastapi import FastAPI, HTTPException, Query
from fastapi.responses import JSONResponse, PlainTextResponse
from youtube_transcript_api import YouTubeTranscriptApi
from youtube_transcript_api._errors import TranscriptsDisabled, NoTranscriptFound
import yt_dlp
import re
import uvicorn
from typing import Optional
import os

# Configuration from environment variables
YOUTUBE_API_HOST = os.getenv("YOUTUBE_API_HOST", "0.0.0.0")
YOUTUBE_API_PORT = int(os.getenv("YOUTUBE_API_PORT", "8000"))

"""
YouTube Data Extractor API - NO VIDEO DOWNLOADS!

This API extracts ONLY metadata, transcripts, and comments.
All yt-dlp configurations use 'skip_download': True to prevent any video file downloads.

Data Sources:
- Transcripts: youtube-transcript-api (unofficial, no quota limits)
- Metadata/Comments: yt-dlp (web scraping, no downloads, no quota limits)
"""

app = FastAPI(
    title="YouTube Data Extractor API",
    description="Extract transcripts, metadata, comments, and more from YouTube videos (NO VIDEO DOWNLOADS)",
    version="2.0.0"
)

def extract_video_id(url: str) -> str:
    """Extract video ID from various YouTube URL formats."""
    patterns = [
        r'(?:v=|\/)([0-9A-Za-z_-]{11}).*',
        r'youtu\.be\/([0-9A-Za-z_-]{11})',
        r'embed\/([0-9A-Za-z_-]{11})',
        r'v\/([0-9A-Za-z_-]{11})'
    ]
    
    for pattern in patterns:
        match = re.search(pattern, url)
        if match:
            return match.group(1)
    
    return None

@app.get("/", tags=["Info"])
async def root():
    """API information and usage guide."""
    return {
        "message": "YouTube Data Extractor API",
        "version": "2.0.0",
        "endpoints": {
            "transcript": "/transcript - Get video transcript (text or JSON with timestamps)",
            "metadata": "/metadata - Get video details (title, description, views, etc.)",
            "full": "/full - Get everything (transcript + metadata + comments)",
            "comments": "/comments - Get video comments",
            "related": "/related - Get related/suggested videos",
            "languages": "/languages - Check available transcript languages"
        },
        "docs": "/docs for interactive API documentation"
    }

@app.get("/transcript", tags=["Transcript"])
async def get_transcript(
    url: str = Query(..., description="YouTube video URL"),
    format: str = Query("text", description="'text' or 'json' with timestamps"),
    lang: str = Query(None, description="Language code (e.g., 'en', 'es')")
):
    """Fetch transcript from a YouTube video."""
    
    video_id = extract_video_id(url)
    if not video_id:
        raise HTTPException(status_code=400, detail="Invalid YouTube URL")
    
    try:
        api = YouTubeTranscriptApi()
        
        if lang:
            transcript = api.fetch(video_id, languages=[lang])
        else:
            transcript = api.fetch(video_id)
        
        if format.lower() == "json":
            result = [
                {
                    "text": segment.text,
                    "start": segment.start,
                    "duration": segment.duration
                }
                for segment in transcript
            ]
            return JSONResponse(content={
                "video_id": video_id,
                "transcript": result,
                "total_segments": len(result)
            })
        else:
            full_text = " ".join([segment.text for segment in transcript])
            return PlainTextResponse(content=full_text)
    
    except TranscriptsDisabled:
        raise HTTPException(status_code=404, detail="Transcripts disabled for this video")
    except NoTranscriptFound:
        raise HTTPException(status_code=404, detail=f"No transcript found for language: {lang if lang else 'default'}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error: {str(e)}")

@app.get("/metadata", tags=["Video Data"])
async def get_metadata(
    url: str = Query(..., description="YouTube video URL")
):
    """
    Get comprehensive video metadata including:
    - Title, description, duration
    - Views, likes, upload date
    - Channel info, tags, categories
    - Thumbnails
    
    NOTE: This ONLY extracts metadata - NO video download!
    """
    
    video_id = extract_video_id(url)
    if not video_id:
        raise HTTPException(status_code=400, detail="Invalid YouTube URL")
    
    try:
        # IMPORTANT: These options prevent any video download
        # We only extract metadata from the YouTube page
        ydl_opts = {
            'quiet': True,
            'no_warnings': True,
            'skip_download': True,  # Never download video files
            'extract_flat': False,  # Get full metadata but no download
            'noplaylist': True,  # Don't process playlists
            'no_color': True
        }
        
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(f'https://www.youtube.com/watch?v={video_id}', download=False)
            
            metadata = {
                "video_id": video_id,
                "title": info.get('title'),
                "description": info.get('description'),
                "duration": info.get('duration'),
                "duration_string": info.get('duration_string'),
                "view_count": info.get('view_count'),
                "like_count": info.get('like_count'),
                "upload_date": info.get('upload_date'),
                "uploader": info.get('uploader'),
                "channel": info.get('channel'),
                "channel_id": info.get('channel_id'),
                "channel_url": info.get('channel_url'),
                "subscriber_count": info.get('channel_follower_count'),
                "categories": info.get('categories', []),
                "tags": info.get('tags', []),
                "thumbnails": info.get('thumbnails', []),
                "webpage_url": info.get('webpage_url')
            }
            
            return JSONResponse(content=metadata)
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error fetching metadata: {str(e)}")

@app.get("/comments", tags=["Video Data"])
async def get_comments(
    url: str = Query(..., description="YouTube video URL"),
    max_comments: int = Query(20, description="Maximum number of comments to fetch (default: 20)")
):
    """
    Fetch comments from a YouTube video.
    
    NOTE: This ONLY extracts comments - NO video download!
    """
    
    video_id = extract_video_id(url)
    if not video_id:
        raise HTTPException(status_code=400, detail="Invalid YouTube URL")
    
    try:
        # IMPORTANT: No video download - only comment extraction
        ydl_opts = {
            'quiet': True,
            'no_warnings': True,
            'skip_download': True,  # Never download video
            'getcomments': True,  # Only fetch comments
            'noplaylist': True,
            'extractor_args': {'youtube': {'max_comments': [str(max_comments)]}}
        }
        
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(f'https://www.youtube.com/watch?v={video_id}', download=False)
            
            comments = []
            raw_comments = info.get('comments', [])
            
            for comment in raw_comments[:max_comments]:
                comments.append({
                    "author": comment.get('author'),
                    "text": comment.get('text'),
                    "like_count": comment.get('like_count'),
                    "timestamp": comment.get('timestamp'),
                    "is_favorited": comment.get('is_favorited', False),
                    "parent": comment.get('parent', 'root')
                })
            
            return JSONResponse(content={
                "video_id": video_id,
                "total_comments": len(comments),
                "comments": comments
            })
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error fetching comments: {str(e)}")

@app.get("/related", tags=["Video Data"])
async def get_related_videos(
    url: str = Query(..., description="YouTube video URL"),
    limit: int = Query(10, description="Number of related videos (default: 10)")
):
    """
    Get related/suggested videos.
    
    NOTE: This ONLY extracts metadata - NO video download!
    """
    
    video_id = extract_video_id(url)
    if not video_id:
        raise HTTPException(status_code=400, detail="Invalid YouTube URL")
    
    try:
        # IMPORTANT: No video download - only metadata
        ydl_opts = {
            'quiet': True,
            'no_warnings': True,
            'skip_download': True,  # Never download video
            'extract_flat': False,
            'noplaylist': True
        }
        
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(f'https://www.youtube.com/watch?v={video_id}', download=False)
            
            # YouTube doesn't always provide related videos in the API
            # But we can get some suggestions from the video info
            related = []
            
            # Try to get related videos from various fields
            if 'entries' in info:
                for entry in info['entries'][:limit]:
                    related.append({
                        "video_id": entry.get('id'),
                        "title": entry.get('title'),
                        "channel": entry.get('channel'),
                        "duration": entry.get('duration'),
                        "view_count": entry.get('view_count')
                    })
            
            return JSONResponse(content={
                "video_id": video_id,
                "related_videos": related,
                "note": "Related videos availability depends on YouTube's API response"
            })
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error fetching related videos: {str(e)}")

@app.get("/full", tags=["Complete Data"])
async def get_full_data(
    url: str = Query(..., description="YouTube video URL"),
    include_transcript: bool = Query(True, description="Include transcript"),
    include_comments: bool = Query(True, description="Include comments"),
    max_comments: int = Query(10, description="Max comments to fetch")
):
    """
    Get complete video data in one request:
    - Full metadata
    - Transcript (if available)
    - Comments (if requested)
    
    NOTE: This ONLY extracts data - NO video download!
    """
    
    video_id = extract_video_id(url)
    if not video_id:
        raise HTTPException(status_code=400, detail="Invalid YouTube URL")
    
    result = {"video_id": video_id}
    
    # Get metadata
    try:
        # IMPORTANT: No video download - only metadata and comments
        ydl_opts = {
            'quiet': True,
            'no_warnings': True,
            'skip_download': True,  # Never download video files
            'getcomments': include_comments,
            'noplaylist': True,
            'extractor_args': {'youtube': {'max_comments': [str(max_comments)]}} if include_comments else {}
        }
        
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(f'https://www.youtube.com/watch?v={video_id}', download=False)
            
            result["metadata"] = {
                "title": info.get('title'),
                "description": info.get('description'),
                "duration": info.get('duration'),
                "view_count": info.get('view_count'),
                "like_count": info.get('like_count'),
                "upload_date": info.get('upload_date'),
                "channel": info.get('channel'),
                "channel_id": info.get('channel_id'),
                "tags": info.get('tags', []),
                "categories": info.get('categories', [])
            }
            
            # Get comments if requested
            if include_comments:
                comments = []
                for comment in info.get('comments', [])[:max_comments]:
                    comments.append({
                        "author": comment.get('author'),
                        "text": comment.get('text'),
                        "like_count": comment.get('like_count')
                    })
                result["comments"] = comments
    
    except Exception as e:
        result["metadata_error"] = str(e)
    
    # Get transcript if requested
    if include_transcript:
        try:
            api = YouTubeTranscriptApi()
            transcript = api.fetch(video_id)
            result["transcript"] = " ".join([segment.text for segment in transcript])
        except:
            result["transcript"] = "Not available"
    
    return JSONResponse(content=result)

@app.get("/languages", tags=["Info"])
async def get_available_languages(
    url: str = Query(..., description="YouTube video URL")
):
    """Get available transcript languages for a video."""
    
    video_id = extract_video_id(url)
    if not video_id:
        raise HTTPException(status_code=400, detail="Invalid YouTube URL")
    
    try:
        api = YouTubeTranscriptApi()
        transcript_list = api.list(video_id)
        
        languages = [
            {
                "language": t.language,
                "language_code": t.language_code,
                "is_generated": t.is_generated,
                "is_translatable": t.is_translatable
            }
            for t in transcript_list
        ]
        
        return JSONResponse(content={
            "video_id": video_id,
            "available_languages": languages
        })
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error: {str(e)}")

@app.get("/health", tags=["Info"])
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "version": "2.0.0"}


if __name__ == "__main__":
    print("🚀 YouTube Data Extractor API Server")
    print("=" * 50)
    print("⚠️  NO VIDEO DOWNLOADS - Metadata & Transcripts Only!")
    print(f"🌐 Server will run on: http://{YOUTUBE_API_HOST}:{YOUTUBE_API_PORT}")
    print("📖 Docs: http://localhost:8000/docs")
    print("\n🎯 Quick Examples:")
    print("  • Metadata: /metadata?url=YOUR_URL")
    print("  • Transcript: /transcript?url=YOUR_URL&format=json")
    print("  • Comments: /comments?url=YOUR_URL&max_comments=20")
    print("  • Everything: /full?url=YOUR_URL")
    print("\n💡 Data Sources:")
    print("  • Transcripts: youtube-transcript-api (unofficial)")
    print("  • Metadata: yt-dlp (web scraping, no downloads)")
    print("=" * 50)
    print("\n")

    uvicorn.run(app, host=YOUTUBE_API_HOST, port=YOUTUBE_API_PORT)
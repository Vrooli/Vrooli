export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 7057,
          "end": 7061
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7062,
              "end": 7067
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7070,
                "end": 7075
              }
            },
            "loc": {
              "start": 7069,
              "end": 7075
            }
          },
          "loc": {
            "start": 7062,
            "end": 7075
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notes",
              "loc": {
                "start": 7083,
                "end": 7088
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_list",
                    "loc": {
                      "start": 7102,
                      "end": 7111
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7099,
                    "end": 7111
                  }
                }
              ],
              "loc": {
                "start": 7089,
                "end": 7117
              }
            },
            "loc": {
              "start": 7083,
              "end": 7117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 7122,
                "end": 7131
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Reminder_full",
                    "loc": {
                      "start": 7145,
                      "end": 7158
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7142,
                    "end": 7158
                  }
                }
              ],
              "loc": {
                "start": 7132,
                "end": 7164
              }
            },
            "loc": {
              "start": 7122,
              "end": 7164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 7169,
                "end": 7178
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Resource_list",
                    "loc": {
                      "start": 7192,
                      "end": 7205
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7189,
                    "end": 7205
                  }
                }
              ],
              "loc": {
                "start": 7179,
                "end": 7211
              }
            },
            "loc": {
              "start": 7169,
              "end": 7211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 7216,
                "end": 7225
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Schedule_list",
                    "loc": {
                      "start": 7239,
                      "end": 7252
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7236,
                    "end": 7252
                  }
                }
              ],
              "loc": {
                "start": 7226,
                "end": 7258
              }
            },
            "loc": {
              "start": 7216,
              "end": 7258
            }
          }
        ],
        "loc": {
          "start": 7077,
          "end": 7262
        }
      },
      "loc": {
        "start": 7057,
        "end": 7262
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 88,
                  "end": 100
                }
              },
              "loc": {
                "start": 88,
                "end": 100
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 114,
                      "end": 130
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 111,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 101,
                "end": 136
              }
            },
            "loc": {
              "start": 81,
              "end": 136
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 148,
                  "end": 152
                }
              },
              "loc": {
                "start": 148,
                "end": 152
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 166,
                      "end": 174
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 174
                  }
                }
              ],
              "loc": {
                "start": 153,
                "end": 180
              }
            },
            "loc": {
              "start": 141,
              "end": 180
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 182
        }
      },
      "loc": {
        "start": 69,
        "end": 182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 183,
          "end": 186
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 193,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 207,
                "end": 216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 216
            }
          }
        ],
        "loc": {
          "start": 187,
          "end": 218
        }
      },
      "loc": {
        "start": 183,
        "end": 218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 250,
          "end": 258
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 265,
                "end": 277
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 288,
                      "end": 290
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 288,
                    "end": 290
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 299,
                      "end": 307
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 299,
                    "end": 307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 316,
                      "end": 327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 316,
                    "end": 327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 336,
                      "end": 340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 336,
                    "end": 340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "text",
                    "loc": {
                      "start": 349,
                      "end": 353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 349,
                    "end": 353
                  }
                }
              ],
              "loc": {
                "start": 278,
                "end": 359
              }
            },
            "loc": {
              "start": 265,
              "end": 359
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 364,
                "end": 366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 364,
              "end": 366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 371,
                "end": 381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 371,
              "end": 381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 386,
                "end": 396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 386,
              "end": 396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 401,
                "end": 409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 401,
              "end": 409
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 414,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 414,
              "end": 423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 428,
                "end": 440
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 428,
              "end": 440
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 445,
                "end": 457
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 445,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 462,
                "end": 474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 462,
              "end": 474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 479,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 493,
                      "end": 503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 493,
                    "end": 503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 512,
                      "end": 519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 512,
                    "end": 519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 528,
                      "end": 537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 528,
                    "end": 537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 546,
                      "end": 555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 546,
                    "end": 555
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 564,
                      "end": 573
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 564,
                    "end": 573
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 582,
                      "end": 588
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 582,
                    "end": 588
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 597,
                      "end": 604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 597,
                    "end": 604
                  }
                }
              ],
              "loc": {
                "start": 483,
                "end": 610
              }
            },
            "loc": {
              "start": 479,
              "end": 610
            }
          }
        ],
        "loc": {
          "start": 259,
          "end": 612
        }
      },
      "loc": {
        "start": 250,
        "end": 612
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 613,
          "end": 615
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 613,
        "end": 615
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 616,
          "end": 626
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 616,
        "end": 626
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 627,
          "end": 637
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 627,
        "end": 637
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 638,
          "end": 647
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 638,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 648,
          "end": 659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 648,
        "end": 659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 660,
          "end": 666
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 676,
                "end": 686
              }
            },
            "directives": [],
            "loc": {
              "start": 673,
              "end": 686
            }
          }
        ],
        "loc": {
          "start": 667,
          "end": 688
        }
      },
      "loc": {
        "start": 660,
        "end": 688
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 689,
          "end": 694
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 708,
                  "end": 720
                }
              },
              "loc": {
                "start": 708,
                "end": 720
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 734,
                      "end": 750
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 731,
                    "end": 750
                  }
                }
              ],
              "loc": {
                "start": 721,
                "end": 756
              }
            },
            "loc": {
              "start": 701,
              "end": 756
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 768,
                  "end": 772
                }
              },
              "loc": {
                "start": 768,
                "end": 772
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 786,
                      "end": 794
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 783,
                    "end": 794
                  }
                }
              ],
              "loc": {
                "start": 773,
                "end": 800
              }
            },
            "loc": {
              "start": 761,
              "end": 800
            }
          }
        ],
        "loc": {
          "start": 695,
          "end": 802
        }
      },
      "loc": {
        "start": 689,
        "end": 802
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 803,
          "end": 814
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 803,
        "end": 814
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 815,
          "end": 829
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 815,
        "end": 829
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 830,
          "end": 835
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 830,
        "end": 835
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 836,
          "end": 845
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 836,
        "end": 845
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 846,
          "end": 850
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 860,
                "end": 868
              }
            },
            "directives": [],
            "loc": {
              "start": 857,
              "end": 868
            }
          }
        ],
        "loc": {
          "start": 851,
          "end": 870
        }
      },
      "loc": {
        "start": 846,
        "end": 870
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 871,
          "end": 885
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 871,
        "end": 885
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 886,
          "end": 891
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 886,
        "end": 891
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 892,
          "end": 895
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 902,
                "end": 911
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 902,
              "end": 911
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 916,
                "end": 927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 916,
              "end": 927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 932,
                "end": 943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 932,
              "end": 943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 948,
                "end": 957
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 948,
              "end": 957
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 962,
                "end": 969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 962,
              "end": 969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 974,
                "end": 982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 974,
              "end": 982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 987,
                "end": 999
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 987,
              "end": 999
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1004,
                "end": 1012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1004,
              "end": 1012
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1017,
                "end": 1025
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1017,
              "end": 1025
            }
          }
        ],
        "loc": {
          "start": 896,
          "end": 1027
        }
      },
      "loc": {
        "start": 892,
        "end": 1027
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1074,
          "end": 1076
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1074,
        "end": 1076
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1077,
          "end": 1088
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1077,
        "end": 1088
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1089,
          "end": 1095
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1089,
        "end": 1095
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1096,
          "end": 1108
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1096,
        "end": 1108
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1109,
          "end": 1112
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 1119,
                "end": 1132
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1119,
              "end": 1132
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1137,
                "end": 1146
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1137,
              "end": 1146
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1151,
                "end": 1162
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1151,
              "end": 1162
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1167,
                "end": 1176
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1167,
              "end": 1176
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1181,
                "end": 1190
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1181,
              "end": 1190
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1195,
                "end": 1202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1195,
              "end": 1202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1207,
                "end": 1219
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1207,
              "end": 1219
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1224,
                "end": 1232
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1224,
              "end": 1232
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1237,
                "end": 1251
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1262,
                      "end": 1264
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1262,
                    "end": 1264
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1273,
                      "end": 1283
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1273,
                    "end": 1283
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1292,
                      "end": 1302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1292,
                    "end": 1302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1311,
                      "end": 1318
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1311,
                    "end": 1318
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1327,
                      "end": 1338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1327,
                    "end": 1338
                  }
                }
              ],
              "loc": {
                "start": 1252,
                "end": 1344
              }
            },
            "loc": {
              "start": 1237,
              "end": 1344
            }
          }
        ],
        "loc": {
          "start": 1113,
          "end": 1346
        }
      },
      "loc": {
        "start": 1109,
        "end": 1346
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1386,
          "end": 1388
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1386,
        "end": 1388
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1389,
          "end": 1399
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1389,
        "end": 1399
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1400,
          "end": 1410
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1400,
        "end": 1410
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1411,
          "end": 1415
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1411,
        "end": 1415
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "description",
        "loc": {
          "start": 1416,
          "end": 1427
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1416,
        "end": 1427
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "dueDate",
        "loc": {
          "start": 1428,
          "end": 1435
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1428,
        "end": 1435
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1436,
          "end": 1441
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1436,
        "end": 1441
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isComplete",
        "loc": {
          "start": 1442,
          "end": 1452
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1442,
        "end": 1452
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderItems",
        "loc": {
          "start": 1453,
          "end": 1466
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1473,
                "end": 1475
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1473,
              "end": 1475
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1480,
                "end": 1490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1480,
              "end": 1490
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1495,
                "end": 1505
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1495,
              "end": 1505
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1510,
                "end": 1514
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1510,
              "end": 1514
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1519,
                "end": 1530
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1519,
              "end": 1530
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1535,
                "end": 1542
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1535,
              "end": 1542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1547,
                "end": 1552
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1547,
              "end": 1552
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1557,
                "end": 1567
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1557,
              "end": 1567
            }
          }
        ],
        "loc": {
          "start": 1467,
          "end": 1569
        }
      },
      "loc": {
        "start": 1453,
        "end": 1569
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderList",
        "loc": {
          "start": 1570,
          "end": 1582
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1589,
                "end": 1591
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1589,
              "end": 1591
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1596,
                "end": 1606
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1596,
              "end": 1606
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1611,
                "end": 1621
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1611,
              "end": 1621
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusMode",
              "loc": {
                "start": 1626,
                "end": 1635
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 1646,
                      "end": 1652
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1667,
                            "end": 1669
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1667,
                          "end": 1669
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 1682,
                            "end": 1687
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1682,
                          "end": 1687
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 1700,
                            "end": 1705
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1700,
                          "end": 1705
                        }
                      }
                    ],
                    "loc": {
                      "start": 1653,
                      "end": 1715
                    }
                  },
                  "loc": {
                    "start": 1646,
                    "end": 1715
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 1724,
                      "end": 1732
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Schedule_common",
                          "loc": {
                            "start": 1750,
                            "end": 1765
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1747,
                          "end": 1765
                        }
                      }
                    ],
                    "loc": {
                      "start": 1733,
                      "end": 1775
                    }
                  },
                  "loc": {
                    "start": 1724,
                    "end": 1775
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1784,
                      "end": 1786
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1784,
                    "end": 1786
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1795,
                      "end": 1799
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1795,
                    "end": 1799
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1808,
                      "end": 1819
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1808,
                    "end": 1819
                  }
                }
              ],
              "loc": {
                "start": 1636,
                "end": 1825
              }
            },
            "loc": {
              "start": 1626,
              "end": 1825
            }
          }
        ],
        "loc": {
          "start": 1583,
          "end": 1827
        }
      },
      "loc": {
        "start": 1570,
        "end": 1827
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1867,
          "end": 1869
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1867,
        "end": 1869
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1870,
          "end": 1875
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1870,
        "end": 1875
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 1876,
          "end": 1880
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1876,
        "end": 1880
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 1881,
          "end": 1888
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1881,
        "end": 1888
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1889,
          "end": 1901
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1908,
                "end": 1910
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1908,
              "end": 1910
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1915,
                "end": 1923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1915,
              "end": 1923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1928,
                "end": 1939
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1928,
              "end": 1939
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1944,
                "end": 1948
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1944,
              "end": 1948
            }
          }
        ],
        "loc": {
          "start": 1902,
          "end": 1950
        }
      },
      "loc": {
        "start": 1889,
        "end": 1950
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1992,
          "end": 1994
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1992,
        "end": 1994
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1995,
          "end": 2005
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1995,
        "end": 2005
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2006,
          "end": 2016
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2006,
        "end": 2016
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 2017,
          "end": 2026
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2017,
        "end": 2026
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 2027,
          "end": 2034
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2027,
        "end": 2034
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 2035,
          "end": 2043
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2035,
        "end": 2043
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 2044,
          "end": 2054
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2061,
                "end": 2063
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2061,
              "end": 2063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 2068,
                "end": 2085
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2068,
              "end": 2085
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 2090,
                "end": 2102
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2090,
              "end": 2102
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 2107,
                "end": 2117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2107,
              "end": 2117
            }
          }
        ],
        "loc": {
          "start": 2055,
          "end": 2119
        }
      },
      "loc": {
        "start": 2044,
        "end": 2119
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 2120,
          "end": 2131
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2138,
                "end": 2140
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2138,
              "end": 2140
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 2145,
                "end": 2159
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2145,
              "end": 2159
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 2164,
                "end": 2172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2164,
              "end": 2172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 2177,
                "end": 2186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2177,
              "end": 2186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 2191,
                "end": 2201
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2191,
              "end": 2201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 2206,
                "end": 2211
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2206,
              "end": 2211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 2216,
                "end": 2223
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2216,
              "end": 2223
            }
          }
        ],
        "loc": {
          "start": 2132,
          "end": 2225
        }
      },
      "loc": {
        "start": 2120,
        "end": 2225
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2265,
          "end": 2271
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 2281,
                "end": 2291
              }
            },
            "directives": [],
            "loc": {
              "start": 2278,
              "end": 2291
            }
          }
        ],
        "loc": {
          "start": 2272,
          "end": 2293
        }
      },
      "loc": {
        "start": 2265,
        "end": 2293
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModes",
        "loc": {
          "start": 2294,
          "end": 2304
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2311,
                "end": 2317
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2328,
                      "end": 2330
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2328,
                    "end": 2330
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "color",
                    "loc": {
                      "start": 2339,
                      "end": 2344
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2339,
                    "end": 2344
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 2353,
                      "end": 2358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2353,
                    "end": 2358
                  }
                }
              ],
              "loc": {
                "start": 2318,
                "end": 2364
              }
            },
            "loc": {
              "start": 2311,
              "end": 2364
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2369,
                "end": 2371
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2369,
              "end": 2371
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2376,
                "end": 2380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2376,
              "end": 2380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2385,
                "end": 2396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2385,
              "end": 2396
            }
          }
        ],
        "loc": {
          "start": 2305,
          "end": 2398
        }
      },
      "loc": {
        "start": 2294,
        "end": 2398
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetings",
        "loc": {
          "start": 2399,
          "end": 2407
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2414,
                "end": 2420
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 2434,
                      "end": 2444
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2431,
                    "end": 2444
                  }
                }
              ],
              "loc": {
                "start": 2421,
                "end": 2450
              }
            },
            "loc": {
              "start": 2414,
              "end": 2450
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2455,
                "end": 2467
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2478,
                      "end": 2480
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2478,
                    "end": 2480
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2489,
                      "end": 2497
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2489,
                    "end": 2497
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2506,
                      "end": 2517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2506,
                    "end": 2517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "link",
                    "loc": {
                      "start": 2526,
                      "end": 2530
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2526,
                    "end": 2530
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2539,
                      "end": 2543
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2539,
                    "end": 2543
                  }
                }
              ],
              "loc": {
                "start": 2468,
                "end": 2549
              }
            },
            "loc": {
              "start": 2455,
              "end": 2549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2554,
                "end": 2556
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2554,
              "end": 2556
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 2561,
                "end": 2583
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2561,
              "end": 2583
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnOrganizationProfile",
              "loc": {
                "start": 2588,
                "end": 2613
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2588,
              "end": 2613
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 2618,
                "end": 2630
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2641,
                      "end": 2643
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2641,
                    "end": 2643
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 2652,
                      "end": 2663
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2652,
                    "end": 2663
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 2672,
                      "end": 2678
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2672,
                    "end": 2678
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 2687,
                      "end": 2699
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2687,
                    "end": 2699
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2708,
                      "end": 2711
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canAddMembers",
                          "loc": {
                            "start": 2726,
                            "end": 2739
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2726,
                          "end": 2739
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 2752,
                            "end": 2761
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2752,
                          "end": 2761
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 2774,
                            "end": 2785
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2774,
                          "end": 2785
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 2798,
                            "end": 2807
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2798,
                          "end": 2807
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2820,
                            "end": 2829
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2820,
                          "end": 2829
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2842,
                            "end": 2849
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2842,
                          "end": 2849
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 2862,
                            "end": 2874
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2862,
                          "end": 2874
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 2887,
                            "end": 2895
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2887,
                          "end": 2895
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 2908,
                            "end": 2922
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2941,
                                  "end": 2943
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2941,
                                "end": 2943
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2960,
                                  "end": 2970
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2960,
                                "end": 2970
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2987,
                                  "end": 2997
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2987,
                                "end": 2997
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3014,
                                  "end": 3021
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3014,
                                "end": 3021
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3038,
                                  "end": 3049
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3038,
                                "end": 3049
                              }
                            }
                          ],
                          "loc": {
                            "start": 2923,
                            "end": 3063
                          }
                        },
                        "loc": {
                          "start": 2908,
                          "end": 3063
                        }
                      }
                    ],
                    "loc": {
                      "start": 2712,
                      "end": 3073
                    }
                  },
                  "loc": {
                    "start": 2708,
                    "end": 3073
                  }
                }
              ],
              "loc": {
                "start": 2631,
                "end": 3079
              }
            },
            "loc": {
              "start": 2618,
              "end": 3079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 3084,
                "end": 3101
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "members",
                    "loc": {
                      "start": 3112,
                      "end": 3119
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3134,
                            "end": 3136
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3134,
                          "end": 3136
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3149,
                            "end": 3159
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3149,
                          "end": 3159
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3172,
                            "end": 3182
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3172,
                          "end": 3182
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3195,
                            "end": 3202
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3195,
                          "end": 3202
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3215,
                            "end": 3226
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3215,
                          "end": 3226
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 3239,
                            "end": 3244
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3263,
                                  "end": 3265
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3263,
                                "end": 3265
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3282,
                                  "end": 3292
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3282,
                                "end": 3292
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3309,
                                  "end": 3319
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3309,
                                "end": 3319
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3336,
                                  "end": 3340
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3336,
                                "end": 3340
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3357,
                                  "end": 3368
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3357,
                                "end": 3368
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 3385,
                                  "end": 3397
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3385,
                                "end": 3397
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "organization",
                                "loc": {
                                  "start": 3414,
                                  "end": 3426
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3449,
                                        "end": 3451
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3449,
                                      "end": 3451
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 3472,
                                        "end": 3483
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3472,
                                      "end": 3483
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 3504,
                                        "end": 3510
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3504,
                                      "end": 3510
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 3531,
                                        "end": 3543
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3531,
                                      "end": 3543
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3564,
                                        "end": 3567
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canAddMembers",
                                            "loc": {
                                              "start": 3594,
                                              "end": 3607
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3594,
                                            "end": 3607
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 3632,
                                              "end": 3641
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3632,
                                            "end": 3641
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 3666,
                                              "end": 3677
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3666,
                                            "end": 3677
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 3702,
                                              "end": 3711
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3702,
                                            "end": 3711
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 3736,
                                              "end": 3745
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3736,
                                            "end": 3745
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 3770,
                                              "end": 3777
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3770,
                                            "end": 3777
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3802,
                                              "end": 3814
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3802,
                                            "end": 3814
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 3839,
                                              "end": 3847
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3839,
                                            "end": 3847
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 3872,
                                              "end": 3886
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 3917,
                                                    "end": 3919
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3917,
                                                  "end": 3919
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3948,
                                                    "end": 3958
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3948,
                                                  "end": 3958
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3987,
                                                    "end": 3997
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3987,
                                                  "end": 3997
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 4026,
                                                    "end": 4033
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4026,
                                                  "end": 4033
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 4062,
                                                    "end": 4073
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4062,
                                                  "end": 4073
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3887,
                                              "end": 4099
                                            }
                                          },
                                          "loc": {
                                            "start": 3872,
                                            "end": 4099
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3568,
                                        "end": 4121
                                      }
                                    },
                                    "loc": {
                                      "start": 3564,
                                      "end": 4121
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3427,
                                  "end": 4139
                                }
                              },
                              "loc": {
                                "start": 3414,
                                "end": 4139
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4156,
                                  "end": 4168
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4191,
                                        "end": 4193
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4191,
                                      "end": 4193
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4214,
                                        "end": 4222
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4214,
                                      "end": 4222
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4243,
                                        "end": 4254
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4243,
                                      "end": 4254
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4169,
                                  "end": 4272
                                }
                              },
                              "loc": {
                                "start": 4156,
                                "end": 4272
                              }
                            }
                          ],
                          "loc": {
                            "start": 3245,
                            "end": 4286
                          }
                        },
                        "loc": {
                          "start": 3239,
                          "end": 4286
                        }
                      }
                    ],
                    "loc": {
                      "start": 3120,
                      "end": 4296
                    }
                  },
                  "loc": {
                    "start": 3112,
                    "end": 4296
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4305,
                      "end": 4307
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4305,
                    "end": 4307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4316,
                      "end": 4326
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4316,
                    "end": 4326
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4335,
                      "end": 4345
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4335,
                    "end": 4345
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4354,
                      "end": 4358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4354,
                    "end": 4358
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 4367,
                      "end": 4378
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4367,
                    "end": 4378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 4387,
                      "end": 4399
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4387,
                    "end": 4399
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 4408,
                      "end": 4420
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4435,
                            "end": 4437
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4435,
                          "end": 4437
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 4450,
                            "end": 4461
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4450,
                          "end": 4461
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 4474,
                            "end": 4480
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4474,
                          "end": 4480
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 4493,
                            "end": 4505
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4493,
                          "end": 4505
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 4518,
                            "end": 4521
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canAddMembers",
                                "loc": {
                                  "start": 4540,
                                  "end": 4553
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4540,
                                "end": 4553
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 4570,
                                  "end": 4579
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4570,
                                "end": 4579
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 4596,
                                  "end": 4607
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4596,
                                "end": 4607
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 4624,
                                  "end": 4633
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4624,
                                "end": 4633
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 4650,
                                  "end": 4659
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4650,
                                "end": 4659
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 4676,
                                  "end": 4683
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4676,
                                "end": 4683
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 4700,
                                  "end": 4712
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4700,
                                "end": 4712
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 4729,
                                  "end": 4737
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4729,
                                "end": 4737
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 4754,
                                  "end": 4768
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4791,
                                        "end": 4793
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4791,
                                      "end": 4793
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4814,
                                        "end": 4824
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4814,
                                      "end": 4824
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4845,
                                        "end": 4855
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4845,
                                      "end": 4855
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 4876,
                                        "end": 4883
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4876,
                                      "end": 4883
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4904,
                                        "end": 4915
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4904,
                                      "end": 4915
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4769,
                                  "end": 4933
                                }
                              },
                              "loc": {
                                "start": 4754,
                                "end": 4933
                              }
                            }
                          ],
                          "loc": {
                            "start": 4522,
                            "end": 4947
                          }
                        },
                        "loc": {
                          "start": 4518,
                          "end": 4947
                        }
                      }
                    ],
                    "loc": {
                      "start": 4421,
                      "end": 4957
                    }
                  },
                  "loc": {
                    "start": 4408,
                    "end": 4957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 4966,
                      "end": 4978
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4993,
                            "end": 4995
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4993,
                          "end": 4995
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5008,
                            "end": 5016
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5008,
                          "end": 5016
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5029,
                            "end": 5040
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5029,
                          "end": 5040
                        }
                      }
                    ],
                    "loc": {
                      "start": 4979,
                      "end": 5050
                    }
                  },
                  "loc": {
                    "start": 4966,
                    "end": 5050
                  }
                }
              ],
              "loc": {
                "start": 3102,
                "end": 5056
              }
            },
            "loc": {
              "start": 3084,
              "end": 5056
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 5061,
                "end": 5075
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5061,
              "end": 5075
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 5080,
                "end": 5092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5080,
              "end": 5092
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5097,
                "end": 5100
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5111,
                      "end": 5120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5111,
                    "end": 5120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 5129,
                      "end": 5138
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5129,
                    "end": 5138
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5147,
                      "end": 5156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5147,
                    "end": 5156
                  }
                }
              ],
              "loc": {
                "start": 5101,
                "end": 5162
              }
            },
            "loc": {
              "start": 5097,
              "end": 5162
            }
          }
        ],
        "loc": {
          "start": 2408,
          "end": 5164
        }
      },
      "loc": {
        "start": 2399,
        "end": 5164
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 5165,
          "end": 5176
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectVersion",
              "loc": {
                "start": 5183,
                "end": 5197
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5208,
                      "end": 5210
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5208,
                    "end": 5210
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 5219,
                      "end": 5229
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5219,
                    "end": 5229
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5238,
                      "end": 5246
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5238,
                    "end": 5246
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5255,
                      "end": 5264
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5255,
                    "end": 5264
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5273,
                      "end": 5285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5273,
                    "end": 5285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5294,
                      "end": 5306
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5294,
                    "end": 5306
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 5315,
                      "end": 5319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5334,
                            "end": 5336
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5334,
                          "end": 5336
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5349,
                            "end": 5358
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5349,
                          "end": 5358
                        }
                      }
                    ],
                    "loc": {
                      "start": 5320,
                      "end": 5368
                    }
                  },
                  "loc": {
                    "start": 5315,
                    "end": 5368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5377,
                      "end": 5389
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5404,
                            "end": 5406
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5404,
                          "end": 5406
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5419,
                            "end": 5427
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5419,
                          "end": 5427
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5440,
                            "end": 5451
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5440,
                          "end": 5451
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5464,
                            "end": 5468
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5464,
                          "end": 5468
                        }
                      }
                    ],
                    "loc": {
                      "start": 5390,
                      "end": 5478
                    }
                  },
                  "loc": {
                    "start": 5377,
                    "end": 5478
                  }
                }
              ],
              "loc": {
                "start": 5198,
                "end": 5484
              }
            },
            "loc": {
              "start": 5183,
              "end": 5484
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5489,
                "end": 5491
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5489,
              "end": 5491
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5496,
                "end": 5505
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5496,
              "end": 5505
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 5510,
                "end": 5529
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5510,
              "end": 5529
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 5534,
                "end": 5549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5534,
              "end": 5549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 5554,
                "end": 5563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5554,
              "end": 5563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 5568,
                "end": 5579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5568,
              "end": 5579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 5584,
                "end": 5595
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5584,
              "end": 5595
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 5600,
                "end": 5604
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5600,
              "end": 5604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 5609,
                "end": 5615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5609,
              "end": 5615
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 5620,
                "end": 5630
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5620,
              "end": 5630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 5635,
                "end": 5647
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 5661,
                      "end": 5677
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5658,
                    "end": 5677
                  }
                }
              ],
              "loc": {
                "start": 5648,
                "end": 5683
              }
            },
            "loc": {
              "start": 5635,
              "end": 5683
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 5688,
                "end": 5692
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 5706,
                      "end": 5714
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5703,
                    "end": 5714
                  }
                }
              ],
              "loc": {
                "start": 5693,
                "end": 5720
              }
            },
            "loc": {
              "start": 5688,
              "end": 5720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5725,
                "end": 5728
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5739,
                      "end": 5748
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5739,
                    "end": 5748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5757,
                      "end": 5766
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5757,
                    "end": 5766
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5775,
                      "end": 5782
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5775,
                    "end": 5782
                  }
                }
              ],
              "loc": {
                "start": 5729,
                "end": 5788
              }
            },
            "loc": {
              "start": 5725,
              "end": 5788
            }
          }
        ],
        "loc": {
          "start": 5177,
          "end": 5790
        }
      },
      "loc": {
        "start": 5165,
        "end": 5790
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 5791,
          "end": 5802
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineVersion",
              "loc": {
                "start": 5809,
                "end": 5823
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5834,
                      "end": 5836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5834,
                    "end": 5836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 5845,
                      "end": 5855
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5845,
                    "end": 5855
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 5864,
                      "end": 5877
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5864,
                    "end": 5877
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5886,
                      "end": 5896
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5886,
                    "end": 5896
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5905,
                      "end": 5914
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5905,
                    "end": 5914
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5923,
                      "end": 5931
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5923,
                    "end": 5931
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5940,
                      "end": 5949
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5940,
                    "end": 5949
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 5958,
                      "end": 5962
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5977,
                            "end": 5979
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5977,
                          "end": 5979
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 5992,
                            "end": 6002
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5992,
                          "end": 6002
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6015,
                            "end": 6024
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6015,
                          "end": 6024
                        }
                      }
                    ],
                    "loc": {
                      "start": 5963,
                      "end": 6034
                    }
                  },
                  "loc": {
                    "start": 5958,
                    "end": 6034
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6043,
                      "end": 6055
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6070,
                            "end": 6072
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6070,
                          "end": 6072
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6085,
                            "end": 6093
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6085,
                          "end": 6093
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6106,
                            "end": 6117
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6106,
                          "end": 6117
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 6130,
                            "end": 6142
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6130,
                          "end": 6142
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6155,
                            "end": 6159
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6155,
                          "end": 6159
                        }
                      }
                    ],
                    "loc": {
                      "start": 6056,
                      "end": 6169
                    }
                  },
                  "loc": {
                    "start": 6043,
                    "end": 6169
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6178,
                      "end": 6190
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6178,
                    "end": 6190
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6199,
                      "end": 6211
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6199,
                    "end": 6211
                  }
                }
              ],
              "loc": {
                "start": 5824,
                "end": 6217
              }
            },
            "loc": {
              "start": 5809,
              "end": 6217
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6222,
                "end": 6224
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6222,
              "end": 6224
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6229,
                "end": 6238
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6229,
              "end": 6238
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6243,
                "end": 6262
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6243,
              "end": 6262
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6267,
                "end": 6282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6267,
              "end": 6282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 6287,
                "end": 6296
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6287,
              "end": 6296
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 6301,
                "end": 6312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6301,
              "end": 6312
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 6317,
                "end": 6328
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6317,
              "end": 6328
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6333,
                "end": 6337
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6333,
              "end": 6337
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 6342,
                "end": 6348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6342,
              "end": 6348
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 6353,
                "end": 6363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6353,
              "end": 6363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 6368,
                "end": 6379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6368,
              "end": 6379
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 6384,
                "end": 6403
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6384,
              "end": 6403
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 6408,
                "end": 6420
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6434,
                      "end": 6450
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6431,
                    "end": 6450
                  }
                }
              ],
              "loc": {
                "start": 6421,
                "end": 6456
              }
            },
            "loc": {
              "start": 6408,
              "end": 6456
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 6461,
                "end": 6465
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 6479,
                      "end": 6487
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6476,
                    "end": 6487
                  }
                }
              ],
              "loc": {
                "start": 6466,
                "end": 6493
              }
            },
            "loc": {
              "start": 6461,
              "end": 6493
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6498,
                "end": 6501
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6512,
                      "end": 6521
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6512,
                    "end": 6521
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6530,
                      "end": 6539
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6530,
                    "end": 6539
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6548,
                      "end": 6555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6548,
                    "end": 6555
                  }
                }
              ],
              "loc": {
                "start": 6502,
                "end": 6561
              }
            },
            "loc": {
              "start": 6498,
              "end": 6561
            }
          }
        ],
        "loc": {
          "start": 5803,
          "end": 6563
        }
      },
      "loc": {
        "start": 5791,
        "end": 6563
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6564,
          "end": 6566
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6564,
        "end": 6566
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6567,
          "end": 6577
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6567,
        "end": 6577
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6578,
          "end": 6588
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6578,
        "end": 6588
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 6589,
          "end": 6598
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6589,
        "end": 6598
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 6599,
          "end": 6606
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6599,
        "end": 6606
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 6607,
          "end": 6615
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6607,
        "end": 6615
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 6616,
          "end": 6626
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6633,
                "end": 6635
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6633,
              "end": 6635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 6640,
                "end": 6657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6640,
              "end": 6657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 6662,
                "end": 6674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6662,
              "end": 6674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 6679,
                "end": 6689
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6679,
              "end": 6689
            }
          }
        ],
        "loc": {
          "start": 6627,
          "end": 6691
        }
      },
      "loc": {
        "start": 6616,
        "end": 6691
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 6692,
          "end": 6703
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6710,
                "end": 6712
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6710,
              "end": 6712
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 6717,
                "end": 6731
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6717,
              "end": 6731
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 6736,
                "end": 6744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6736,
              "end": 6744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 6749,
                "end": 6758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6749,
              "end": 6758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 6763,
                "end": 6773
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6763,
              "end": 6773
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 6778,
                "end": 6783
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6778,
              "end": 6783
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 6788,
                "end": 6795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6788,
              "end": 6795
            }
          }
        ],
        "loc": {
          "start": 6704,
          "end": 6797
        }
      },
      "loc": {
        "start": 6692,
        "end": 6797
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6827,
          "end": 6829
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6827,
        "end": 6829
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6830,
          "end": 6840
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6830,
        "end": 6840
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 6841,
          "end": 6844
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6841,
        "end": 6844
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6845,
          "end": 6854
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6845,
        "end": 6854
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6855,
          "end": 6867
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6874,
                "end": 6876
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6874,
              "end": 6876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6881,
                "end": 6889
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6881,
              "end": 6889
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 6894,
                "end": 6905
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6894,
              "end": 6905
            }
          }
        ],
        "loc": {
          "start": 6868,
          "end": 6907
        }
      },
      "loc": {
        "start": 6855,
        "end": 6907
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6908,
          "end": 6911
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 6918,
                "end": 6923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6918,
              "end": 6923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6928,
                "end": 6940
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6928,
              "end": 6940
            }
          }
        ],
        "loc": {
          "start": 6912,
          "end": 6942
        }
      },
      "loc": {
        "start": 6908,
        "end": 6942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6973,
          "end": 6975
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6973,
        "end": 6975
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 6976,
          "end": 6987
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6976,
        "end": 6987
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 6988,
          "end": 6994
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6988,
        "end": 6994
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 6995,
          "end": 7000
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6995,
        "end": 7000
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7001,
          "end": 7005
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7001,
        "end": 7005
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7006,
          "end": 7018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7006,
        "end": 7018
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 88,
                        "end": 100
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 100
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 114,
                            "end": 130
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 111,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 101,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 136
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 148,
                        "end": 152
                      }
                    },
                    "loc": {
                      "start": 148,
                      "end": 152
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 166,
                            "end": 174
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 163,
                          "end": 174
                        }
                      }
                    ],
                    "loc": {
                      "start": 153,
                      "end": 180
                    }
                  },
                  "loc": {
                    "start": 141,
                    "end": 180
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 182
              }
            },
            "loc": {
              "start": 69,
              "end": 182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 183,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 193,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 207,
                      "end": 216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 216
                  }
                }
              ],
              "loc": {
                "start": 187,
                "end": 218
              }
            },
            "loc": {
              "start": 183,
              "end": 218
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 220
        }
      },
      "loc": {
        "start": 1,
        "end": 220
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 230,
          "end": 239
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 243,
            "end": 247
          }
        },
        "loc": {
          "start": 243,
          "end": 247
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 250,
                "end": 258
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 265,
                      "end": 277
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 288,
                            "end": 290
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 288,
                          "end": 290
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 299,
                            "end": 307
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 299,
                          "end": 307
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 316,
                            "end": 327
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 316,
                          "end": 327
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 336,
                            "end": 340
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 336,
                          "end": 340
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 349,
                            "end": 353
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 349,
                          "end": 353
                        }
                      }
                    ],
                    "loc": {
                      "start": 278,
                      "end": 359
                    }
                  },
                  "loc": {
                    "start": 265,
                    "end": 359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 364,
                      "end": 366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 364,
                    "end": 366
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 371,
                      "end": 381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 371,
                    "end": 381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 386,
                      "end": 396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 386,
                    "end": 396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 401,
                      "end": 409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 401,
                    "end": 409
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 414,
                      "end": 423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 414,
                    "end": 423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 428,
                      "end": 440
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 428,
                    "end": 440
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 445,
                      "end": 457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 445,
                    "end": 457
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 462,
                      "end": 474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 462,
                    "end": 474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 479,
                      "end": 482
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 493,
                            "end": 503
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 493,
                          "end": 503
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 512,
                            "end": 519
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 512,
                          "end": 519
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 528,
                            "end": 537
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 528,
                          "end": 537
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 546,
                            "end": 555
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 546,
                          "end": 555
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 564,
                            "end": 573
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 564,
                          "end": 573
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 582,
                            "end": 588
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 582,
                          "end": 588
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 597,
                            "end": 604
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 597,
                          "end": 604
                        }
                      }
                    ],
                    "loc": {
                      "start": 483,
                      "end": 610
                    }
                  },
                  "loc": {
                    "start": 479,
                    "end": 610
                  }
                }
              ],
              "loc": {
                "start": 259,
                "end": 612
              }
            },
            "loc": {
              "start": 250,
              "end": 612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 613,
                "end": 615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 613,
              "end": 615
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 616,
                "end": 626
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 616,
              "end": 626
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 627,
                "end": 637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 627,
              "end": 637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 638,
                "end": 647
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 638,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 648,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 648,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 660,
                "end": 666
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 676,
                      "end": 686
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 673,
                    "end": 686
                  }
                }
              ],
              "loc": {
                "start": 667,
                "end": 688
              }
            },
            "loc": {
              "start": 660,
              "end": 688
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 689,
                "end": 694
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 708,
                        "end": 720
                      }
                    },
                    "loc": {
                      "start": 708,
                      "end": 720
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 734,
                            "end": 750
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 731,
                          "end": 750
                        }
                      }
                    ],
                    "loc": {
                      "start": 721,
                      "end": 756
                    }
                  },
                  "loc": {
                    "start": 701,
                    "end": 756
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 768,
                        "end": 772
                      }
                    },
                    "loc": {
                      "start": 768,
                      "end": 772
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 786,
                            "end": 794
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 783,
                          "end": 794
                        }
                      }
                    ],
                    "loc": {
                      "start": 773,
                      "end": 800
                    }
                  },
                  "loc": {
                    "start": 761,
                    "end": 800
                  }
                }
              ],
              "loc": {
                "start": 695,
                "end": 802
              }
            },
            "loc": {
              "start": 689,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 803,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 803,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 815,
                "end": 829
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 815,
              "end": 829
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 830,
                "end": 835
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 830,
              "end": 835
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 836,
                "end": 845
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 836,
              "end": 845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 846,
                "end": 850
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 860,
                      "end": 868
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 857,
                    "end": 868
                  }
                }
              ],
              "loc": {
                "start": 851,
                "end": 870
              }
            },
            "loc": {
              "start": 846,
              "end": 870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 871,
                "end": 885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 871,
              "end": 885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 886,
                "end": 891
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 886,
              "end": 891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 892,
                "end": 895
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 902,
                      "end": 911
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 902,
                    "end": 911
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 916,
                      "end": 927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 916,
                    "end": 927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 932,
                      "end": 943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 932,
                    "end": 943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 948,
                      "end": 957
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 948,
                    "end": 957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 962,
                      "end": 969
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 962,
                    "end": 969
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 974,
                      "end": 982
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 974,
                    "end": 982
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 987,
                      "end": 999
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 987,
                    "end": 999
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1004,
                      "end": 1012
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1004,
                    "end": 1012
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1017,
                      "end": 1025
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1017,
                    "end": 1025
                  }
                }
              ],
              "loc": {
                "start": 896,
                "end": 1027
              }
            },
            "loc": {
              "start": 892,
              "end": 1027
            }
          }
        ],
        "loc": {
          "start": 248,
          "end": 1029
        }
      },
      "loc": {
        "start": 221,
        "end": 1029
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 1039,
          "end": 1055
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 1059,
            "end": 1071
          }
        },
        "loc": {
          "start": 1059,
          "end": 1071
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1074,
                "end": 1076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1074,
              "end": 1076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1077,
                "end": 1088
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1077,
              "end": 1088
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1089,
                "end": 1095
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1089,
              "end": 1095
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1096,
                "end": 1108
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1096,
              "end": 1108
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1109,
                "end": 1112
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1119,
                      "end": 1132
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1119,
                    "end": 1132
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1137,
                      "end": 1146
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1137,
                    "end": 1146
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1151,
                      "end": 1162
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1151,
                    "end": 1162
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1167,
                      "end": 1176
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1167,
                    "end": 1176
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1181,
                      "end": 1190
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1181,
                    "end": 1190
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1195,
                      "end": 1202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1195,
                    "end": 1202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1207,
                      "end": 1219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1207,
                    "end": 1219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1224,
                      "end": 1232
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1224,
                    "end": 1232
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1237,
                      "end": 1251
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1262,
                            "end": 1264
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1262,
                          "end": 1264
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1273,
                            "end": 1283
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1273,
                          "end": 1283
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1292,
                            "end": 1302
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1292,
                          "end": 1302
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1311,
                            "end": 1318
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1311,
                          "end": 1318
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1327,
                            "end": 1338
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1327,
                          "end": 1338
                        }
                      }
                    ],
                    "loc": {
                      "start": 1252,
                      "end": 1344
                    }
                  },
                  "loc": {
                    "start": 1237,
                    "end": 1344
                  }
                }
              ],
              "loc": {
                "start": 1113,
                "end": 1346
              }
            },
            "loc": {
              "start": 1109,
              "end": 1346
            }
          }
        ],
        "loc": {
          "start": 1072,
          "end": 1348
        }
      },
      "loc": {
        "start": 1030,
        "end": 1348
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 1358,
          "end": 1371
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 1375,
            "end": 1383
          }
        },
        "loc": {
          "start": 1375,
          "end": 1383
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1386,
                "end": 1388
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1386,
              "end": 1388
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1389,
                "end": 1399
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1389,
              "end": 1399
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1400,
                "end": 1410
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1400,
              "end": 1410
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1411,
                "end": 1415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1411,
              "end": 1415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1416,
                "end": 1427
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1416,
              "end": 1427
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1428,
                "end": 1435
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1428,
              "end": 1435
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1436,
                "end": 1441
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1436,
              "end": 1441
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1442,
                "end": 1452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1442,
              "end": 1452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 1453,
                "end": 1466
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1473,
                      "end": 1475
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1473,
                    "end": 1475
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1480,
                      "end": 1490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1480,
                    "end": 1490
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1495,
                      "end": 1505
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1495,
                    "end": 1505
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1510,
                      "end": 1514
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1510,
                    "end": 1514
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1519,
                      "end": 1530
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1519,
                    "end": 1530
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 1535,
                      "end": 1542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1535,
                    "end": 1542
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 1547,
                      "end": 1552
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1547,
                    "end": 1552
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1557,
                      "end": 1567
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1557,
                    "end": 1567
                  }
                }
              ],
              "loc": {
                "start": 1467,
                "end": 1569
              }
            },
            "loc": {
              "start": 1453,
              "end": 1569
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 1570,
                "end": 1582
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1589,
                      "end": 1591
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1589,
                    "end": 1591
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1596,
                      "end": 1606
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1596,
                    "end": 1606
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1611,
                      "end": 1621
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1611,
                    "end": 1621
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 1626,
                      "end": 1635
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 1646,
                            "end": 1652
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1667,
                                  "end": 1669
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1667,
                                "end": 1669
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 1682,
                                  "end": 1687
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1682,
                                "end": 1687
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 1700,
                                  "end": 1705
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1700,
                                "end": 1705
                              }
                            }
                          ],
                          "loc": {
                            "start": 1653,
                            "end": 1715
                          }
                        },
                        "loc": {
                          "start": 1646,
                          "end": 1715
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 1724,
                            "end": 1732
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 1750,
                                  "end": 1765
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1747,
                                "end": 1765
                              }
                            }
                          ],
                          "loc": {
                            "start": 1733,
                            "end": 1775
                          }
                        },
                        "loc": {
                          "start": 1724,
                          "end": 1775
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1784,
                            "end": 1786
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1784,
                          "end": 1786
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1795,
                            "end": 1799
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1795,
                          "end": 1799
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1808,
                            "end": 1819
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1808,
                          "end": 1819
                        }
                      }
                    ],
                    "loc": {
                      "start": 1636,
                      "end": 1825
                    }
                  },
                  "loc": {
                    "start": 1626,
                    "end": 1825
                  }
                }
              ],
              "loc": {
                "start": 1583,
                "end": 1827
              }
            },
            "loc": {
              "start": 1570,
              "end": 1827
            }
          }
        ],
        "loc": {
          "start": 1384,
          "end": 1829
        }
      },
      "loc": {
        "start": 1349,
        "end": 1829
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1839,
          "end": 1852
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1856,
            "end": 1864
          }
        },
        "loc": {
          "start": 1856,
          "end": 1864
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1867,
                "end": 1869
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1867,
              "end": 1869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1870,
                "end": 1875
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1870,
              "end": 1875
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1876,
                "end": 1880
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1876,
              "end": 1880
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1881,
                "end": 1888
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1881,
              "end": 1888
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1889,
                "end": 1901
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1908,
                      "end": 1910
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1908,
                    "end": 1910
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1915,
                      "end": 1923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1915,
                    "end": 1923
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1928,
                      "end": 1939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1928,
                    "end": 1939
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1944,
                      "end": 1948
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1944,
                    "end": 1948
                  }
                }
              ],
              "loc": {
                "start": 1902,
                "end": 1950
              }
            },
            "loc": {
              "start": 1889,
              "end": 1950
            }
          }
        ],
        "loc": {
          "start": 1865,
          "end": 1952
        }
      },
      "loc": {
        "start": 1830,
        "end": 1952
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 1962,
          "end": 1977
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1981,
            "end": 1989
          }
        },
        "loc": {
          "start": 1981,
          "end": 1989
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1992,
                "end": 1994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1992,
              "end": 1994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1995,
                "end": 2005
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1995,
              "end": 2005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2006,
                "end": 2016
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2006,
              "end": 2016
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 2017,
                "end": 2026
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2017,
              "end": 2026
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2027,
                "end": 2034
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2027,
              "end": 2034
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2035,
                "end": 2043
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2035,
              "end": 2043
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2044,
                "end": 2054
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2061,
                      "end": 2063
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2061,
                    "end": 2063
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2068,
                      "end": 2085
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2068,
                    "end": 2085
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2090,
                      "end": 2102
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2090,
                    "end": 2102
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2107,
                      "end": 2117
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2107,
                    "end": 2117
                  }
                }
              ],
              "loc": {
                "start": 2055,
                "end": 2119
              }
            },
            "loc": {
              "start": 2044,
              "end": 2119
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2120,
                "end": 2131
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2138,
                      "end": 2140
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2138,
                    "end": 2140
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2145,
                      "end": 2159
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2145,
                    "end": 2159
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2164,
                      "end": 2172
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2164,
                    "end": 2172
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2177,
                      "end": 2186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2177,
                    "end": 2186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2191,
                      "end": 2201
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2191,
                    "end": 2201
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2206,
                      "end": 2211
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2206,
                    "end": 2211
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2216,
                      "end": 2223
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2216,
                    "end": 2223
                  }
                }
              ],
              "loc": {
                "start": 2132,
                "end": 2225
              }
            },
            "loc": {
              "start": 2120,
              "end": 2225
            }
          }
        ],
        "loc": {
          "start": 1990,
          "end": 2227
        }
      },
      "loc": {
        "start": 1953,
        "end": 2227
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 2237,
          "end": 2250
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2254,
            "end": 2262
          }
        },
        "loc": {
          "start": 2254,
          "end": 2262
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2265,
                "end": 2271
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 2281,
                      "end": 2291
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2278,
                    "end": 2291
                  }
                }
              ],
              "loc": {
                "start": 2272,
                "end": 2293
              }
            },
            "loc": {
              "start": 2265,
              "end": 2293
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 2294,
                "end": 2304
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 2311,
                      "end": 2317
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2328,
                            "end": 2330
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2328,
                          "end": 2330
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 2339,
                            "end": 2344
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2339,
                          "end": 2344
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2353,
                            "end": 2358
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2353,
                          "end": 2358
                        }
                      }
                    ],
                    "loc": {
                      "start": 2318,
                      "end": 2364
                    }
                  },
                  "loc": {
                    "start": 2311,
                    "end": 2364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2369,
                      "end": 2371
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2369,
                    "end": 2371
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2376,
                      "end": 2380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2376,
                    "end": 2380
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2385,
                      "end": 2396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2385,
                    "end": 2396
                  }
                }
              ],
              "loc": {
                "start": 2305,
                "end": 2398
              }
            },
            "loc": {
              "start": 2294,
              "end": 2398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetings",
              "loc": {
                "start": 2399,
                "end": 2407
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 2414,
                      "end": 2420
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Label_list",
                          "loc": {
                            "start": 2434,
                            "end": 2444
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2431,
                          "end": 2444
                        }
                      }
                    ],
                    "loc": {
                      "start": 2421,
                      "end": 2450
                    }
                  },
                  "loc": {
                    "start": 2414,
                    "end": 2450
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2455,
                      "end": 2467
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2478,
                            "end": 2480
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2478,
                          "end": 2480
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2489,
                            "end": 2497
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2489,
                          "end": 2497
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2506,
                            "end": 2517
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2506,
                          "end": 2517
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 2526,
                            "end": 2530
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2526,
                          "end": 2530
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2539,
                            "end": 2543
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2539,
                          "end": 2543
                        }
                      }
                    ],
                    "loc": {
                      "start": 2468,
                      "end": 2549
                    }
                  },
                  "loc": {
                    "start": 2455,
                    "end": 2549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2554,
                      "end": 2556
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2554,
                    "end": 2556
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 2561,
                      "end": 2583
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2561,
                    "end": 2583
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnOrganizationProfile",
                    "loc": {
                      "start": 2588,
                      "end": 2613
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2588,
                    "end": 2613
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 2618,
                      "end": 2630
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2641,
                            "end": 2643
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2641,
                          "end": 2643
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 2652,
                            "end": 2663
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2652,
                          "end": 2663
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 2672,
                            "end": 2678
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2672,
                          "end": 2678
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 2687,
                            "end": 2699
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2687,
                          "end": 2699
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 2708,
                            "end": 2711
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canAddMembers",
                                "loc": {
                                  "start": 2726,
                                  "end": 2739
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2726,
                                "end": 2739
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 2752,
                                  "end": 2761
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2752,
                                "end": 2761
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 2774,
                                  "end": 2785
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2774,
                                "end": 2785
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 2798,
                                  "end": 2807
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2798,
                                "end": 2807
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 2820,
                                  "end": 2829
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2820,
                                "end": 2829
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 2842,
                                  "end": 2849
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2842,
                                "end": 2849
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 2862,
                                  "end": 2874
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2862,
                                "end": 2874
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 2887,
                                  "end": 2895
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2887,
                                "end": 2895
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 2908,
                                  "end": 2922
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2941,
                                        "end": 2943
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2941,
                                      "end": 2943
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2960,
                                        "end": 2970
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2960,
                                      "end": 2970
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 2987,
                                        "end": 2997
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2987,
                                      "end": 2997
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 3014,
                                        "end": 3021
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3014,
                                      "end": 3021
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3038,
                                        "end": 3049
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3038,
                                      "end": 3049
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2923,
                                  "end": 3063
                                }
                              },
                              "loc": {
                                "start": 2908,
                                "end": 3063
                              }
                            }
                          ],
                          "loc": {
                            "start": 2712,
                            "end": 3073
                          }
                        },
                        "loc": {
                          "start": 2708,
                          "end": 3073
                        }
                      }
                    ],
                    "loc": {
                      "start": 2631,
                      "end": 3079
                    }
                  },
                  "loc": {
                    "start": 2618,
                    "end": 3079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 3084,
                      "end": 3101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "members",
                          "loc": {
                            "start": 3112,
                            "end": 3119
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3134,
                                  "end": 3136
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3134,
                                "end": 3136
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3149,
                                  "end": 3159
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3149,
                                "end": 3159
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3172,
                                  "end": 3182
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3172,
                                "end": 3182
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3195,
                                  "end": 3202
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3195,
                                "end": 3202
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3215,
                                  "end": 3226
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3215,
                                "end": 3226
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 3239,
                                  "end": 3244
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3263,
                                        "end": 3265
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3263,
                                      "end": 3265
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3282,
                                        "end": 3292
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3282,
                                      "end": 3292
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3309,
                                        "end": 3319
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3309,
                                      "end": 3319
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3336,
                                        "end": 3340
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3336,
                                      "end": 3340
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3357,
                                        "end": 3368
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3357,
                                      "end": 3368
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 3385,
                                        "end": 3397
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3385,
                                      "end": 3397
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "organization",
                                      "loc": {
                                        "start": 3414,
                                        "end": 3426
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3449,
                                              "end": 3451
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3449,
                                            "end": 3451
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 3472,
                                              "end": 3483
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3472,
                                            "end": 3483
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 3504,
                                              "end": 3510
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3504,
                                            "end": 3510
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 3531,
                                              "end": 3543
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3531,
                                            "end": 3543
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 3564,
                                              "end": 3567
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canAddMembers",
                                                  "loc": {
                                                    "start": 3594,
                                                    "end": 3607
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3594,
                                                  "end": 3607
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 3632,
                                                    "end": 3641
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3632,
                                                  "end": 3641
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 3666,
                                                    "end": 3677
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3666,
                                                  "end": 3677
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 3702,
                                                    "end": 3711
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3702,
                                                  "end": 3711
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 3736,
                                                    "end": 3745
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3736,
                                                  "end": 3745
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 3770,
                                                    "end": 3777
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3770,
                                                  "end": 3777
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 3802,
                                                    "end": 3814
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3802,
                                                  "end": 3814
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 3839,
                                                    "end": 3847
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3839,
                                                  "end": 3847
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 3872,
                                                    "end": 3886
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 3917,
                                                          "end": 3919
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3917,
                                                        "end": 3919
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 3948,
                                                          "end": 3958
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3948,
                                                        "end": 3958
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 3987,
                                                          "end": 3997
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3987,
                                                        "end": 3997
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 4026,
                                                          "end": 4033
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4026,
                                                        "end": 4033
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 4062,
                                                          "end": 4073
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4062,
                                                        "end": 4073
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 3887,
                                                    "end": 4099
                                                  }
                                                },
                                                "loc": {
                                                  "start": 3872,
                                                  "end": 4099
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3568,
                                              "end": 4121
                                            }
                                          },
                                          "loc": {
                                            "start": 3564,
                                            "end": 4121
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3427,
                                        "end": 4139
                                      }
                                    },
                                    "loc": {
                                      "start": 3414,
                                      "end": 4139
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4156,
                                        "end": 4168
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4191,
                                              "end": 4193
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4191,
                                            "end": 4193
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4214,
                                              "end": 4222
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4214,
                                            "end": 4222
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4243,
                                              "end": 4254
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4243,
                                            "end": 4254
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4169,
                                        "end": 4272
                                      }
                                    },
                                    "loc": {
                                      "start": 4156,
                                      "end": 4272
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3245,
                                  "end": 4286
                                }
                              },
                              "loc": {
                                "start": 3239,
                                "end": 4286
                              }
                            }
                          ],
                          "loc": {
                            "start": 3120,
                            "end": 4296
                          }
                        },
                        "loc": {
                          "start": 3112,
                          "end": 4296
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4305,
                            "end": 4307
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4305,
                          "end": 4307
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4316,
                            "end": 4326
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4316,
                          "end": 4326
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4335,
                            "end": 4345
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4335,
                          "end": 4345
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4354,
                            "end": 4358
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4354,
                          "end": 4358
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 4367,
                            "end": 4378
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4367,
                          "end": 4378
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 4387,
                            "end": 4399
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4387,
                          "end": 4399
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "organization",
                          "loc": {
                            "start": 4408,
                            "end": 4420
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4435,
                                  "end": 4437
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4435,
                                "end": 4437
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 4450,
                                  "end": 4461
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4450,
                                "end": 4461
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 4474,
                                  "end": 4480
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4474,
                                "end": 4480
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 4493,
                                  "end": 4505
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4493,
                                "end": 4505
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 4518,
                                  "end": 4521
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canAddMembers",
                                      "loc": {
                                        "start": 4540,
                                        "end": 4553
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4540,
                                      "end": 4553
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 4570,
                                        "end": 4579
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4570,
                                      "end": 4579
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 4596,
                                        "end": 4607
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4596,
                                      "end": 4607
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 4624,
                                        "end": 4633
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4624,
                                      "end": 4633
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 4650,
                                        "end": 4659
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4650,
                                      "end": 4659
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 4676,
                                        "end": 4683
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4676,
                                      "end": 4683
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 4700,
                                        "end": 4712
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4700,
                                      "end": 4712
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 4729,
                                        "end": 4737
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4729,
                                      "end": 4737
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 4754,
                                        "end": 4768
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4791,
                                              "end": 4793
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4791,
                                            "end": 4793
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4814,
                                              "end": 4824
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4814,
                                            "end": 4824
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4845,
                                              "end": 4855
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4845,
                                            "end": 4855
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 4876,
                                              "end": 4883
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4876,
                                            "end": 4883
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 4904,
                                              "end": 4915
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4904,
                                            "end": 4915
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4769,
                                        "end": 4933
                                      }
                                    },
                                    "loc": {
                                      "start": 4754,
                                      "end": 4933
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4522,
                                  "end": 4947
                                }
                              },
                              "loc": {
                                "start": 4518,
                                "end": 4947
                              }
                            }
                          ],
                          "loc": {
                            "start": 4421,
                            "end": 4957
                          }
                        },
                        "loc": {
                          "start": 4408,
                          "end": 4957
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 4966,
                            "end": 4978
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4993,
                                  "end": 4995
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4993,
                                "end": 4995
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5008,
                                  "end": 5016
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5008,
                                "end": 5016
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5029,
                                  "end": 5040
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5029,
                                "end": 5040
                              }
                            }
                          ],
                          "loc": {
                            "start": 4979,
                            "end": 5050
                          }
                        },
                        "loc": {
                          "start": 4966,
                          "end": 5050
                        }
                      }
                    ],
                    "loc": {
                      "start": 3102,
                      "end": 5056
                    }
                  },
                  "loc": {
                    "start": 3084,
                    "end": 5056
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 5061,
                      "end": 5075
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5061,
                    "end": 5075
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 5080,
                      "end": 5092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5080,
                    "end": 5092
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5097,
                      "end": 5100
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5111,
                            "end": 5120
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5111,
                          "end": 5120
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 5129,
                            "end": 5138
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5129,
                          "end": 5138
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5147,
                            "end": 5156
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5147,
                          "end": 5156
                        }
                      }
                    ],
                    "loc": {
                      "start": 5101,
                      "end": 5162
                    }
                  },
                  "loc": {
                    "start": 5097,
                    "end": 5162
                  }
                }
              ],
              "loc": {
                "start": 2408,
                "end": 5164
              }
            },
            "loc": {
              "start": 2399,
              "end": 5164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 5165,
                "end": 5176
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectVersion",
                    "loc": {
                      "start": 5183,
                      "end": 5197
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5208,
                            "end": 5210
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5208,
                          "end": 5210
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 5219,
                            "end": 5229
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5219,
                          "end": 5229
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 5238,
                            "end": 5246
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5238,
                          "end": 5246
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5255,
                            "end": 5264
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5255,
                          "end": 5264
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 5273,
                            "end": 5285
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5273,
                          "end": 5285
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 5294,
                            "end": 5306
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5294,
                          "end": 5306
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 5315,
                            "end": 5319
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5334,
                                  "end": 5336
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5334,
                                "end": 5336
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 5349,
                                  "end": 5358
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5349,
                                "end": 5358
                              }
                            }
                          ],
                          "loc": {
                            "start": 5320,
                            "end": 5368
                          }
                        },
                        "loc": {
                          "start": 5315,
                          "end": 5368
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5377,
                            "end": 5389
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5404,
                                  "end": 5406
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5404,
                                "end": 5406
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5419,
                                  "end": 5427
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5419,
                                "end": 5427
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5440,
                                  "end": 5451
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5440,
                                "end": 5451
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 5464,
                                  "end": 5468
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5464,
                                "end": 5468
                              }
                            }
                          ],
                          "loc": {
                            "start": 5390,
                            "end": 5478
                          }
                        },
                        "loc": {
                          "start": 5377,
                          "end": 5478
                        }
                      }
                    ],
                    "loc": {
                      "start": 5198,
                      "end": 5484
                    }
                  },
                  "loc": {
                    "start": 5183,
                    "end": 5484
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5489,
                      "end": 5491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5489,
                    "end": 5491
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5496,
                      "end": 5505
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5496,
                    "end": 5505
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 5510,
                      "end": 5529
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5510,
                    "end": 5529
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 5534,
                      "end": 5549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5534,
                    "end": 5549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 5554,
                      "end": 5563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5554,
                    "end": 5563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 5568,
                      "end": 5579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5568,
                    "end": 5579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 5584,
                      "end": 5595
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5584,
                    "end": 5595
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5600,
                      "end": 5604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5600,
                    "end": 5604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 5609,
                      "end": 5615
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5609,
                    "end": 5615
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 5620,
                      "end": 5630
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5620,
                    "end": 5630
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 5635,
                      "end": 5647
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 5661,
                            "end": 5677
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5658,
                          "end": 5677
                        }
                      }
                    ],
                    "loc": {
                      "start": 5648,
                      "end": 5683
                    }
                  },
                  "loc": {
                    "start": 5635,
                    "end": 5683
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 5688,
                      "end": 5692
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 5706,
                            "end": 5714
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5703,
                          "end": 5714
                        }
                      }
                    ],
                    "loc": {
                      "start": 5693,
                      "end": 5720
                    }
                  },
                  "loc": {
                    "start": 5688,
                    "end": 5720
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5725,
                      "end": 5728
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5739,
                            "end": 5748
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5739,
                          "end": 5748
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5757,
                            "end": 5766
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5757,
                          "end": 5766
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 5775,
                            "end": 5782
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5775,
                          "end": 5782
                        }
                      }
                    ],
                    "loc": {
                      "start": 5729,
                      "end": 5788
                    }
                  },
                  "loc": {
                    "start": 5725,
                    "end": 5788
                  }
                }
              ],
              "loc": {
                "start": 5177,
                "end": 5790
              }
            },
            "loc": {
              "start": 5165,
              "end": 5790
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 5791,
                "end": 5802
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineVersion",
                    "loc": {
                      "start": 5809,
                      "end": 5823
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5834,
                            "end": 5836
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5834,
                          "end": 5836
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 5845,
                            "end": 5855
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5845,
                          "end": 5855
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 5864,
                            "end": 5877
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5864,
                          "end": 5877
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 5886,
                            "end": 5896
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5886,
                          "end": 5896
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 5905,
                            "end": 5914
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5905,
                          "end": 5914
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 5923,
                            "end": 5931
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5923,
                          "end": 5931
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5940,
                            "end": 5949
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5940,
                          "end": 5949
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 5958,
                            "end": 5962
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5977,
                                  "end": 5979
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5977,
                                "end": 5979
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 5992,
                                  "end": 6002
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5992,
                                "end": 6002
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6015,
                                  "end": 6024
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6015,
                                "end": 6024
                              }
                            }
                          ],
                          "loc": {
                            "start": 5963,
                            "end": 6034
                          }
                        },
                        "loc": {
                          "start": 5958,
                          "end": 6034
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6043,
                            "end": 6055
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6070,
                                  "end": 6072
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6070,
                                "end": 6072
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6085,
                                  "end": 6093
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6085,
                                "end": 6093
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6106,
                                  "end": 6117
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6106,
                                "end": 6117
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 6130,
                                  "end": 6142
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6130,
                                "end": 6142
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 6155,
                                  "end": 6159
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6155,
                                "end": 6159
                              }
                            }
                          ],
                          "loc": {
                            "start": 6056,
                            "end": 6169
                          }
                        },
                        "loc": {
                          "start": 6043,
                          "end": 6169
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6178,
                            "end": 6190
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6178,
                          "end": 6190
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6199,
                            "end": 6211
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6199,
                          "end": 6211
                        }
                      }
                    ],
                    "loc": {
                      "start": 5824,
                      "end": 6217
                    }
                  },
                  "loc": {
                    "start": 5809,
                    "end": 6217
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6222,
                      "end": 6224
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6222,
                    "end": 6224
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6229,
                      "end": 6238
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6229,
                    "end": 6238
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6243,
                      "end": 6262
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6243,
                    "end": 6262
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6267,
                      "end": 6282
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6267,
                    "end": 6282
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 6287,
                      "end": 6296
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6287,
                    "end": 6296
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6301,
                      "end": 6312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6301,
                    "end": 6312
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 6317,
                      "end": 6328
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6317,
                    "end": 6328
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6333,
                      "end": 6337
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6333,
                    "end": 6337
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 6342,
                      "end": 6348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6342,
                    "end": 6348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 6353,
                      "end": 6363
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6353,
                    "end": 6363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 6368,
                      "end": 6379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6368,
                    "end": 6379
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 6384,
                      "end": 6403
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6384,
                    "end": 6403
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 6408,
                      "end": 6420
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6434,
                            "end": 6450
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6431,
                          "end": 6450
                        }
                      }
                    ],
                    "loc": {
                      "start": 6421,
                      "end": 6456
                    }
                  },
                  "loc": {
                    "start": 6408,
                    "end": 6456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 6461,
                      "end": 6465
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6479,
                            "end": 6487
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6476,
                          "end": 6487
                        }
                      }
                    ],
                    "loc": {
                      "start": 6466,
                      "end": 6493
                    }
                  },
                  "loc": {
                    "start": 6461,
                    "end": 6493
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6498,
                      "end": 6501
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6512,
                            "end": 6521
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6512,
                          "end": 6521
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6530,
                            "end": 6539
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6530,
                          "end": 6539
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6548,
                            "end": 6555
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6548,
                          "end": 6555
                        }
                      }
                    ],
                    "loc": {
                      "start": 6502,
                      "end": 6561
                    }
                  },
                  "loc": {
                    "start": 6498,
                    "end": 6561
                  }
                }
              ],
              "loc": {
                "start": 5803,
                "end": 6563
              }
            },
            "loc": {
              "start": 5791,
              "end": 6563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6564,
                "end": 6566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6564,
              "end": 6566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6567,
                "end": 6577
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6567,
              "end": 6577
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6578,
                "end": 6588
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6578,
              "end": 6588
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 6589,
                "end": 6598
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6589,
              "end": 6598
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 6599,
                "end": 6606
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6599,
              "end": 6606
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 6607,
                "end": 6615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6607,
              "end": 6615
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 6616,
                "end": 6626
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6633,
                      "end": 6635
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6633,
                    "end": 6635
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 6640,
                      "end": 6657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6640,
                    "end": 6657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 6662,
                      "end": 6674
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6662,
                    "end": 6674
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 6679,
                      "end": 6689
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6679,
                    "end": 6689
                  }
                }
              ],
              "loc": {
                "start": 6627,
                "end": 6691
              }
            },
            "loc": {
              "start": 6616,
              "end": 6691
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 6692,
                "end": 6703
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6710,
                      "end": 6712
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6710,
                    "end": 6712
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 6717,
                      "end": 6731
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6717,
                    "end": 6731
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 6736,
                      "end": 6744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6736,
                    "end": 6744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 6749,
                      "end": 6758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6749,
                    "end": 6758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 6763,
                      "end": 6773
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6763,
                    "end": 6773
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 6778,
                      "end": 6783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6778,
                    "end": 6783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 6788,
                      "end": 6795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6788,
                    "end": 6795
                  }
                }
              ],
              "loc": {
                "start": 6704,
                "end": 6797
              }
            },
            "loc": {
              "start": 6692,
              "end": 6797
            }
          }
        ],
        "loc": {
          "start": 2263,
          "end": 6799
        }
      },
      "loc": {
        "start": 2228,
        "end": 6799
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 6809,
          "end": 6817
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 6821,
            "end": 6824
          }
        },
        "loc": {
          "start": 6821,
          "end": 6824
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6827,
                "end": 6829
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6827,
              "end": 6829
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6830,
                "end": 6840
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6830,
              "end": 6840
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 6841,
                "end": 6844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6841,
              "end": 6844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6845,
                "end": 6854
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6845,
              "end": 6854
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6855,
                "end": 6867
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6874,
                      "end": 6876
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6874,
                    "end": 6876
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6881,
                      "end": 6889
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6881,
                    "end": 6889
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6894,
                      "end": 6905
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6894,
                    "end": 6905
                  }
                }
              ],
              "loc": {
                "start": 6868,
                "end": 6907
              }
            },
            "loc": {
              "start": 6855,
              "end": 6907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6908,
                "end": 6911
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 6918,
                      "end": 6923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6918,
                    "end": 6923
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6928,
                      "end": 6940
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6928,
                    "end": 6940
                  }
                }
              ],
              "loc": {
                "start": 6912,
                "end": 6942
              }
            },
            "loc": {
              "start": 6908,
              "end": 6942
            }
          }
        ],
        "loc": {
          "start": 6825,
          "end": 6944
        }
      },
      "loc": {
        "start": 6800,
        "end": 6944
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 6954,
          "end": 6962
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 6966,
            "end": 6970
          }
        },
        "loc": {
          "start": 6966,
          "end": 6970
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6973,
                "end": 6975
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6973,
              "end": 6975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 6976,
                "end": 6987
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6976,
              "end": 6987
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 6988,
                "end": 6994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6988,
              "end": 6994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 6995,
                "end": 7000
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6995,
              "end": 7000
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7001,
                "end": 7005
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7001,
              "end": 7005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7006,
                "end": 7018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7006,
              "end": 7018
            }
          }
        ],
        "loc": {
          "start": 6971,
          "end": 7020
        }
      },
      "loc": {
        "start": 6945,
        "end": 7020
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "home",
      "loc": {
        "start": 7028,
        "end": 7032
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7034,
              "end": 7039
            }
          },
          "loc": {
            "start": 7033,
            "end": 7039
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "HomeInput",
              "loc": {
                "start": 7041,
                "end": 7050
              }
            },
            "loc": {
              "start": 7041,
              "end": 7050
            }
          },
          "loc": {
            "start": 7041,
            "end": 7051
          }
        },
        "directives": [],
        "loc": {
          "start": 7033,
          "end": 7051
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "home",
            "loc": {
              "start": 7057,
              "end": 7061
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7062,
                  "end": 7067
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7070,
                    "end": 7075
                  }
                },
                "loc": {
                  "start": 7069,
                  "end": 7075
                }
              },
              "loc": {
                "start": 7062,
                "end": 7075
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "notes",
                  "loc": {
                    "start": 7083,
                    "end": 7088
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Note_list",
                        "loc": {
                          "start": 7102,
                          "end": 7111
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7099,
                        "end": 7111
                      }
                    }
                  ],
                  "loc": {
                    "start": 7089,
                    "end": 7117
                  }
                },
                "loc": {
                  "start": 7083,
                  "end": 7117
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 7122,
                    "end": 7131
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Reminder_full",
                        "loc": {
                          "start": 7145,
                          "end": 7158
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7142,
                        "end": 7158
                      }
                    }
                  ],
                  "loc": {
                    "start": 7132,
                    "end": 7164
                  }
                },
                "loc": {
                  "start": 7122,
                  "end": 7164
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 7169,
                    "end": 7178
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Resource_list",
                        "loc": {
                          "start": 7192,
                          "end": 7205
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7189,
                        "end": 7205
                      }
                    }
                  ],
                  "loc": {
                    "start": 7179,
                    "end": 7211
                  }
                },
                "loc": {
                  "start": 7169,
                  "end": 7211
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 7216,
                    "end": 7225
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Schedule_list",
                        "loc": {
                          "start": 7239,
                          "end": 7252
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7236,
                        "end": 7252
                      }
                    }
                  ],
                  "loc": {
                    "start": 7226,
                    "end": 7258
                  }
                },
                "loc": {
                  "start": 7216,
                  "end": 7258
                }
              }
            ],
            "loc": {
              "start": 7077,
              "end": 7262
            }
          },
          "loc": {
            "start": 7057,
            "end": 7262
          }
        }
      ],
      "loc": {
        "start": 7053,
        "end": 7264
      }
    },
    "loc": {
      "start": 7022,
      "end": 7264
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
